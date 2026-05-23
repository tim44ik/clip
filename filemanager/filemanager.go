package filemanager

import (
	"bytes"
	"clip/encrypter"
	"clip/errors"
	"clip/locales"
	"clip/modules"
	outputprocessor "clip/outputProcessor"
	"clip/reporter"
	st "clip/storage"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

type FileManager struct {
	window        fyne.Window
	lang          string
	path          string
	profileExists bool
	modules       *modules.ClipModules
}

type Script struct {
	Name    string
	Path    string
	Content string
	Chosen  bool
}

func NewFileManager(lang, path string, window fyne.Window, profileExists bool) *FileManager {
	return &FileManager{window: window, lang: lang, path: path, profileExists: profileExists}
}

func (f *FileManager) GetProfileType(errCh chan<- error, skip bool, callback func(st.Encoder)) {
	var enc st.Encoder
	if skip {
		enc = st.NewEncoder(filepath.Ext(f.path))
		if enc == nil {
			errCh <- errors.New(errSavingProfile)
			return
		}
		callback(enc)
		return
	}
	var selected string
	radio := widget.NewRadioGroup([]string{".json"}, func(value string) {
		selected = value
	})

	d := dialog.NewCustomConfirm(
		locales.T(f.lang, "choose_output_type"),
		locales.T(f.lang, "ok"),
		locales.T(f.lang, "cancel"),
		container.NewPadded(
			container.NewBorder(radio, nil, nil, nil),
		),
		func(confirm bool) {
			if confirm {
				enc = st.NewEncoder(selected)
				if enc == nil {
					errCh <- errors.New(errSavingProfile)
					return
				}
				callback(enc)
			}
		},
		f.window,
	)

	d.Resize(fyne.NewSize(300, 200))
	d.Show()
}

func (f *FileManager) SaveProfile(errCh chan<- error, enc encrypter.Encrypter, t st.Encoder, mods *modules.ClipModules, callback func(encrypter.Encrypter, bool, string)) {
	save := func(path string, enc encrypter.Encrypter, password string) {
		if err, _ := t.Encode(mods, path); err != nil {
			errCh <- err
		}
		if enc != nil {
			err := enc.Encrypt(path, password)
			if err != nil {
				errCh <- err
			}
		}

	}

	askPasswordAndSave := func(path string, enc encrypter.Encrypter) {
		f.GetPassword(errCh, func(p string) {
			save(path, enc, p)
		})
	}
	switch f.profileExists {
	case true:
		if enc == nil {
			save(f.path, enc, "")
			return
		}
		askPasswordAndSave(f.path, enc)
	case false:
		f.GetEncryptionType(func(e encrypter.Encrypter) {
			if e == nil {
				f.SaveProfileAs(errCh, "", e, t, mods, callback)
				return
			}
			f.GetPassword(errCh, func(p string) {
				f.SaveProfileAs(errCh, p, e, t, mods, callback)
			})

		})
	}

}

func (f *FileManager) SaveProfileAs(errCh chan<- error, password string, enc encrypter.Encrypter, t st.Encoder, mods *modules.ClipModules, callback func(encrypter.Encrypter, bool, string)) {
	filesavedialog := dialog.NewFileSave(
		func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				errCh <- errors.New(errSavingProfile)
				return
			}

			if writer == nil {
				return
			}

			f.path = writer.URI().Path()
			writer.Close()

			err, path := t.Encode(mods, f.path)
			if err != nil {
				errCh <- err
				return
			}

			if filepath.Ext(f.path) != t.GetFileType() {
				defer os.Remove(f.path)
			}

			f.profileExists = true
			if enc != nil {
				err := enc.Encrypt(path, password)
				if err != nil {
					errCh <- err
					return
				}
			}
			callback(enc, f.profileExists, path)
		}, f.window)

	filesavedialog.SetFilter(storage.NewExtensionFileFilter([]string{t.GetFileType()}))

	filesavedialog.Resize(fyne.NewSize(900, 500))

	filesavedialog.Show()
}

func (f *FileManager) LoadProfile(errCh chan<- error, callback func(modules.ClipModules, string, encrypter.Encrypter)) {
	var mods modules.ClipModules
	var d st.Decoder
	fileOpenDialog := dialog.NewFileOpen(
		func(reader fyne.URIReadCloser, err error) {

			if err != nil {
				errCh <- errors.New(errReadingProfile)
				return
			}

			if reader == nil {
				return
			}

			defer reader.Close()

			f.path = reader.URI().Path()

			fileData, err := os.ReadFile(f.path)
			if err != nil {
				errCh <- errors.New(errReadingProfile)
				return
			}

			d = st.NewDecoder(reader.URI().Extension())
			if d == nil {
				errCh <- errors.New(errReadingProfile)
			}

			switch f.EncryptionChecker(fileData) {
			case false:
				err = d.Decode(&mods, fileData)
				if err != nil {
					errCh <- err
					return
				}

				callback(mods, f.path, nil)
				return
			case true:
				f.GetPassword(errCh, func(password string) {
					enc, decrypted, err := f.DecryptFile(fileData, password)
					if err != nil {
						errCh <- err
						return
					}

					err = d.Decode(&mods, decrypted)
					if err != nil {
						errCh <- err
						return
					}
					callback(mods, f.path, enc)
				})
			}
		},
		f.window,
	)

	fileOpenDialog.SetFilter(
		storage.NewExtensionFileFilter([]string{".json"}),
	)
	fileOpenDialog.Resize(fyne.NewSize(900, 500))
	fileOpenDialog.Show()
}

func (f *FileManager) EncryptionChecker(fileData []byte) bool {
	var isEncrypted bool
	if len(fileData) >= 7 && bytes.Equal(fileData[:4], []byte("CFG1")) {
		flags := fileData[5]
		if flags&0x01 != 0 {
			isEncrypted = true
		}
	}
	return isEncrypted
}

func (f *FileManager) GetReportType(errCh chan<- error, db outputprocessor.DB, callback func(reporter.Reporter, outputprocessor.DB)) {
	var selected string
	var rp reporter.Reporter
	radio := widget.NewRadioGroup([]string{".pdf"}, func(value string) {
		selected = value
	})

	d := dialog.NewCustomConfirm(
		locales.T(f.lang, "choose_output_type"),
		locales.T(f.lang, "ok"),
		locales.T(f.lang, "cancel"),
		container.NewPadded(
			container.NewBorder(radio, nil, nil, nil),
		),
		func(confirm bool) {
			if confirm {
				rp = reporter.NewReporter(selected)
				if rp == nil {
					errCh <- errors.New(errReportType)
					return
				}
				callback(rp, db)
			}
		},
		f.window,
	)

	d.Resize(fyne.NewSize(300, 200))
	d.Show()
}

func (f *FileManager) GetDBType(ctx context.Context, callback func(outputprocessor.DB)) {
	var selected string
	var chosenDB outputprocessor.DB
	radio := widget.NewRadioGroup([]string{"NVD"}, func(value string) {
		selected = value
	})

	d := dialog.NewCustomConfirm(
		locales.T(f.lang, "choose_vuln_db"),
		locales.T(f.lang, "ok"),
		locales.T(f.lang, "cancel"),
		container.NewPadded(
			container.NewBorder(radio, nil, nil, nil),
		),
		func(confirm bool) {
			if confirm {
				chosenDB = outputprocessor.NewDB(selected, ctx)
				if chosenDB == nil {
					return
				}
				callback(chosenDB)
			}
		},
		f.window,
	)

	d.Resize(fyne.NewSize(200, 200))
	d.Show()
}

func (f *FileManager) ReportCreationWindow(
	db outputprocessor.DB,
	r reporter.Reporter,
	callback func(string)) {
	filesaveDialog := dialog.NewFileSave(
		func(writer fyne.URIWriteCloser, err error) {
			go func() {
				if err != nil || writer == nil {
					return
				}

				path := writer.URI().Path()
				ext := r.GetFileType()

				if filepath.Base(path) == ext {
					path = time.Now().Format("02.01.2006 15-04-05") + ext
				}

				go callback(path)
			}()
		}, f.window)
	filesaveDialog.SetFilter(storage.NewExtensionFileFilter([]string{".pdf"}))
	filesaveDialog.Resize(fyne.NewSize(900, 500))
	fyne.Do(func() { filesaveDialog.Show() })
}

func (f *FileManager) GetEncryptionType(callback func(encrypter.Encrypter)) {
	var selected string
	var enc encrypter.Encrypter
	radio := widget.NewRadioGroup([]string{"AES256", locales.T(f.lang, "no_encryption")}, func(value string) {
		selected = value
	})

	d := dialog.NewCustomConfirm(
		locales.T(f.lang, "choose_encryption"),
		locales.T(f.lang, "ok"),
		locales.T(f.lang, "cancel"),
		container.NewPadded(
			container.NewBorder(radio, nil, nil, nil),
		),
		func(confirm bool) {
			if confirm {
				enc = encrypter.NewEncrypter(selected)
				callback(enc)
			}
		},
		f.window,
	)

	d.Resize(fyne.NewSize(300, 200))
	d.Show()
}

func (f *FileManager) GetPassword(errCh chan<- error, callback func(string)) {
	input := widget.NewPasswordEntry()
	passwordDialog := dialog.NewCustomConfirm(
		locales.T(f.lang, "enter_password"),
		locales.T(f.lang, "ok"),
		locales.T(f.lang, "cancel"),
		container.NewPadded(
			container.NewBorder(
				input, nil, nil, nil,
			),
		), func(b bool) {
			if b {
				if input.Text != "" {
					callback(input.Text)
				} else {
					errCh <- errors.New(errPasswordNotProvided)
					return
				}

			}
		}, f.window)
	passwordDialog.Resize(fyne.NewSize(300, 100))
	passwordDialog.Show()
}

func (f *FileManager) DecryptFile(data []byte, password string) (encrypter.Encrypter, []byte, error) {
	if len(data) < 7 {
		return nil, nil, errors.New(errInvalidData)
	}

	if !bytes.Equal(data[:4], []byte("CFG1")) {
		return nil, data, nil
	}

	version := data[4]
	if version != 1 {
		return nil, nil, errors.New(errInvalidData)
	}

	flags := data[5]
	cipherID := data[6]
	payload := data[7:]

	if flags&0x01 == 0 {
		return nil, payload, nil
	}
	var cipher encrypter.Decrypter

	cipher = encrypter.NewDecrypter(int(cipherID))
	if cipher == nil {
		return nil, nil, errors.New(errUnknownCipher)
	}
	data, err := cipher.Decrypt(payload, password)
	return cipher.(encrypter.Encrypter), data, err

}

func (f *FileManager) LoadScripts(errCh chan<- error, callback func([]*Script)) {
	dialog.ShowFolderOpen(func(list fyne.ListableURI, err error) {
		if err != nil || list == nil {
			errCh <- errors.New(errOpeningFolder)
			return
		}

		files, err := list.List()
		if err != nil {
			errCh <- errors.New(errListingFiles)
			return
		}

		scripts := make([]*Script, 0, len(files))
		for _, f := range files {

			ext := strings.ToLower(filepath.Ext(f.Name()))
			if ext != ".sh" && ext != ".txt" && ext != ".bat" && ext != ".ps" {
				continue
			}

			scripts = append(scripts, &Script{
				Name: f.Name(),
				Path: f.Path(),
			})
		}
		callback(scripts)
	}, f.window)
}

func (f *FileManager) OpenScriptPicker(
	files []*Script,
	callback func(name, content string),
) {
	var picker dialog.Dialog

	list := widget.NewList(
		func() int {
			return len(files)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewCheck("", nil),
				widget.NewLabel(""),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			row := o.(*fyne.Container)

			check := row.Objects[0].(*widget.Check)
			label := row.Objects[1].(*widget.Label)

			label.SetText(files[i].Name)

			check.OnChanged = nil
			check.SetChecked(files[i].Chosen)

			check.OnChanged = func(v bool) {
				files[i].Chosen = v
			}
		},
	)

	selectAllBtn := widget.NewButton(locales.T(f.lang, "select_all"), func() {
		for _, file := range files {
			file.Chosen = true
		}
		list.Refresh()
	})

	clearBtn := widget.NewButton(locales.T(f.lang, "clear"), func() {
		for _, file := range files {
			file.Chosen = false
		}
		list.Refresh()
	})

	controls := container.NewHBox(selectAllBtn, clearBtn)
	content := container.NewBorder(nil, controls, nil, nil, list)

	picker = dialog.NewCustomConfirm(
		locales.T(f.lang, "choose_scripts"),
		locales.T(f.lang, "load_chosen"),
		locales.T(f.lang, "cancel"),
		content,
		func(b bool) {
			if b {
				for _, file := range files {
					if !file.Chosen {
						continue
					}

					uri, err := storage.ParseURI("file://" + file.Path)
					if err != nil {
						continue
					}

					rc, err := storage.Reader(uri)
					if err != nil {
						continue
					}

					data, err := io.ReadAll(rc)
					rc.Close()
					if err != nil {
						continue
					}

					callback(file.Name, string(data))
				}

				picker.Hide()
			}
		},
		f.window,
	)

	picker.Resize(fyne.NewSize(600, 400))
	picker.Show()
}
