package filemanager

import (
	"bytes"
	"clip/encrypter"
	"clip/errors"
	"clip/modules"
	outputprocessor "clip/outputProcessor"
	"clip/reporter"
	st "clip/storage"
	"context"
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
	langslice     []string
	path          string
	profileExists bool
	modules       *modules.ClipModules
}

func NewFileManager(window fyne.Window, langslice []string, path string, profileExists bool) *FileManager {
	return &FileManager{window: window, langslice: langslice, path: path, profileExists: profileExists}
}

func (f *FileManager) GetProfileType(skip bool, callback func(st.Encoder)) {
	if skip {
		callback(st.NewJson())
		return
	}
	var selected string
	radio := widget.NewRadioGroup([]string{"JSON"}, func(value string) {
		selected = value
	})

	d := dialog.NewCustomConfirm(
		f.langslice[41],
		f.langslice[23],
		f.langslice[24],
		container.NewPadded(
			container.NewBorder(radio, nil, nil, nil),
		),
		func(confirm bool) {
			if confirm {
				switch selected {
				case "JSON":
					callback(st.NewJson())
					return
				default:
					dialog.ShowError(errors.UniversalError{ErrorText: f.langslice[40]}, f.window)
					return
				}
			}
		},
		f.window,
	)

	d.Resize(fyne.NewSize(300, 200))
	d.Show()
}

func (f *FileManager) SaveProfile(enc encrypter.Encrypter, t st.Encoder, mods *modules.ClipModules, callback func(encrypter.Encrypter, bool, string)) {
	save := func(path string, enc encrypter.Encrypter, password string) {
		if err, _ := t.Encode(mods, path); err != nil {
			dialog.ShowError(
				errors.UniversalError{ErrorText: f.langslice[40]},
				f.window,
			)
		}
		if enc != nil {
			err := enc.Encrypt(path, password)
			if err != nil {
				dialog.ShowError(
					errors.UniversalError{ErrorText: f.langslice[44]},
					f.window,
				)
			}
		}

	}

	askPasswordAndSave := func(path string, enc encrypter.Encrypter) {
		f.GetPassword(func(p string) {
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
				f.SaveProfileAs("", e, t, mods, callback)
			}
			f.GetPassword(func(p string) {
				f.SaveProfileAs(p, e, t, mods, callback)
			})

		})
	}

}

func (f *FileManager) SaveProfileAs(password string, enc encrypter.Encrypter, t st.Encoder, mods *modules.ClipModules, callback func(encrypter.Encrypter, bool, string)) {
	filesavedialog := dialog.NewFileSave(
		func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				return
			}

			if writer == nil {
				return
			}

			f.path = writer.URI().Path()
			writer.Close()

			err, path := t.Encode(mods, f.path)
			if err != nil {
				dialog.ShowError(errors.UniversalError{ErrorText: f.langslice[40]}, f.window)
				return
			}

			if filepath.Ext(f.path) != t.GetFileType() {
				defer os.Remove(f.path)
			}

			f.profileExists = true
			if enc != nil {
				err := enc.Encrypt(path, password)
				if err != nil {
					dialog.ShowError(
						errors.UniversalError{
							ErrorText: f.langslice[44],
						},
						f.window,
					)
					return
				}
			}
			callback(enc, f.profileExists, path)
		}, f.window)

	filesavedialog.SetFilter(storage.NewExtensionFileFilter([]string{t.GetFileType()}))

	filesavedialog.Resize(fyne.NewSize(900, 500))

	filesavedialog.Show()
}

func (f *FileManager) LoadProfile(callback func(modules.ClipModules, string, encrypter.Encrypter)) {
	var mods modules.ClipModules
	var d st.Decoder
	fileOpenDialog := dialog.NewFileOpen(
		func(reader fyne.URIReadCloser, err error) {

			if err != nil {
				dialog.ShowError(errors.UniversalError{ErrorText: f.langslice[39]}, f.window)
				return
			}

			if reader == nil {
				return
			}

			defer reader.Close()

			f.path = reader.URI().Path()

			fileData, err := os.ReadFile(f.path)
			if err != nil {
				dialog.ShowError(errors.UniversalError{ErrorText: f.langslice[39]}, f.window)
				return
			}

			ext := filepath.Ext(f.path)

			switch ext {
			case ".json":
				d = st.NewJson()
			default:
				dialog.ShowError(errors.UniversalError{ErrorText: f.langslice[39]}, f.window)
				return
			}
			switch f.EncryptionChecker(fileData) {
			case false:
				err = d.Decode(&mods, fileData)
				if err != nil {
					dialog.ShowError(errors.UniversalError{ErrorText: f.langslice[39]}, f.window)
					return
				}

				callback(mods, f.path, nil)
				return
			case true:
				f.GetPassword(func(password string) {
					if password == "" {
						dialog.ShowError(
							errors.UniversalError{
								ErrorText: f.langslice[46],
							},
							f.window,
						)
						return
					}
					enc, decrypted, err := f.DecryptFile(fileData, password)
					if err != nil {
						dialog.ShowError(err, f.window)
						return
					}

					err = d.Decode(&mods, decrypted)
					if err != nil {
						dialog.ShowError(errors.UniversalError{ErrorText: f.langslice[39]}, f.window)
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

func (f *FileManager) GetReportType(callback func(reporter.Reporter)) {
	var selected string
	radio := widget.NewRadioGroup([]string{"PDF"}, func(value string) {
		selected = value
	})

	d := dialog.NewCustomConfirm(
		f.langslice[41],
		f.langslice[23],
		f.langslice[24],
		container.NewPadded(
			container.NewBorder(radio, nil, nil, nil),
		),
		func(confirm bool) {
			if confirm {
				switch selected {
				case "PDF":
					callback(reporter.NewPDF())
					return
				default:
					dialog.ShowError(errors.UniversalError{ErrorText: f.langslice[28]}, f.window)
					return
				}
			}
		},
		f.window,
	)

	d.Resize(fyne.NewSize(300, 200))
	d.Show()
}

func (f *FileManager) GetDBType(ctx context.Context, callback func(outputprocessor.DB)) {
	var selected string
	radio := widget.NewRadioGroup([]string{"NVD"}, func(value string) {
		selected = value
	})

	d := dialog.NewCustomConfirm(
		f.langslice[48],
		f.langslice[23],
		f.langslice[24],
		container.NewPadded(
			container.NewBorder(radio, nil, nil, nil),
		),
		func(confirm bool) {
			if confirm {
				switch selected {
				case "NVD":
					callback(outputprocessor.NewNVDClient(ctx))
				}
			}
		},
		f.window,
	)

	d.Resize(fyne.NewSize(300, 200))
	d.Show()
}

func (f *FileManager) ReportСreationWindow(
	db outputprocessor.DB,
	r reporter.Reporter,
	makePDFFor []*modules.Module) {
	filesaveDialog := dialog.NewFileSave(
		func(writer fyne.URIWriteCloser, err error) {
			go func() {
				if err != nil || writer == nil {
					return
				}

				path := writer.URI().Path()
				ext := r.GetFileType()
				if filepath.Ext(path) != ext {
					path = strings.TrimSuffix(path, filepath.Ext(path))
					path += ext
					defer os.Remove(path)
				}

				if filepath.Base(path) == ext {
					path = time.Now().Format("02.01.2006 15-04-05") + ext
				}

				progressChan := make(chan float64)
				defer close(progressChan)

				errChan := make(chan error)
				defer close(errChan)

				go r.CreateReport(db, makePDFFor, path, progressChan, errChan)

				progressBar := widget.NewProgressBar()
				progressWindow := dialog.NewCustomWithoutButtons(
					"Creating PDF",
					progressBar,
					f.window,
				)
				progressWindow.Show()
				for progressBar.Value < 1 {
					select {
					case p := <-progressChan:
						fyne.DoAndWait(func() {
							progressBar.SetValue(p)
						})
					case <-errChan:
						progressWindow.Hide()
						fyne.DoAndWait(func() { dialog.ShowError(errors.UniversalError{ErrorText: f.langslice[28]}, f.window) })
						return
					}
				}
				fyne.Do(func() { progressWindow.Hide() })
			}()
		}, f.window)
	filesaveDialog.SetFilter(storage.NewExtensionFileFilter([]string{".pdf"}))
	filesaveDialog.Resize(fyne.NewSize(900, 500))
	fyne.Do(func() { filesaveDialog.Show() })
}

func (f *FileManager) GetEncryptionType(callback func(encrypter.Encrypter)) {
	var selected string
	var enc encrypter.Encrypter
	radio := widget.NewRadioGroup([]string{"AES-GCM", f.langslice[47]}, func(value string) {
		selected = value
	})

	d := dialog.NewCustomConfirm(
		f.langslice[42],
		f.langslice[23],
		f.langslice[24],
		container.NewPadded(
			container.NewBorder(radio, nil, nil, nil),
		),
		func(confirm bool) {
			if confirm {
				switch selected {
				case "AES-GCM":
					enc = encrypter.NewAES256SCRYPT()
				}
				callback(enc)
			}
		},
		f.window,
	)

	d.Resize(fyne.NewSize(300, 200))
	d.Show()
}

func (f *FileManager) GetPassword(callback func(string)) {
	input := widget.NewPasswordEntry()
	passwordDialog := dialog.NewCustomConfirm(
		f.langslice[43],
		f.langslice[23],
		f.langslice[24],
		container.NewPadded(
			container.NewBorder(
				input, nil, nil, nil,
			),
		), func(b bool) {
			if b {
				if input.Text != "" {
					callback(input.Text)
				} else {
					dialog.ShowError(
						errors.UniversalError{
							ErrorText: f.langslice[44],
						},
						f.window,
					)
				}

			}
		}, f.window)
	passwordDialog.Resize(fyne.NewSize(300, 100))
	passwordDialog.Show()
}

func (f *FileManager) DecryptFile(data []byte, password string) (encrypter.Encrypter, []byte, error) {
	if len(data) < 7 {
		return nil, nil, errors.UniversalError{ErrorText: f.langslice[44]}
	}

	if !bytes.Equal(data[:4], []byte("CFG1")) {
		return nil, data, nil
	}

	version := data[4]
	if version != 1 {
		return nil, nil, errors.UniversalError{ErrorText: f.langslice[44]}
	}

	flags := data[5]
	cipherID := data[6]
	payload := data[7:]

	if flags&0x01 == 0 {
		return nil, payload, nil
	}
	var cipher encrypter.Decrypter
	switch cipherID {
	case 1:
		cipher = encrypter.NewAES256SCRYPT()
		data, err := cipher.Decrypt(payload, password)
		return cipher.(encrypter.Encrypter), data, err
	default:
		return nil, nil, errors.UniversalError{ErrorText: f.langslice[44]}
	}
}
