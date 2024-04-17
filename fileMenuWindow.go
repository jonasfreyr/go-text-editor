package main

import (
	gc "github.com/rthornton128/goncurses"
	"os"
	"path/filepath"
	"sort"
)

type FileMenuWindow struct {
	menuWindow *MenuWindow
}

func NewFileMenuWindow(y, x, h, w int) (*FileMenuWindow, error) {
	menuWindow, err := NewMenuWindow(y, x, h, w)
	if err != nil {
		return nil, err
	}

	mw := &FileMenuWindow{
		menuWindow: menuWindow,
	}
	return mw, nil
}

func (w *FileMenuWindow) getFiles(currentPath string) ([]MenuItem, error) {
	config := GetEditorConfig()

	files, err := os.ReadDir(currentPath)
	if err != nil {
		return nil, err
	}

	directoryNames := make([]string, 0)
	fileNames := make([]string, 0)
	for _, file := range files {
		fileInfo, err := os.Stat(filepath.Join(currentPath, file.Name()))
		if err != nil {
			continue
		}

		if fileInfo.IsDir() {
			directoryNames = append(directoryNames, file.Name())
		} else {
			fileNames = append(fileNames, file.Name())
		}
	}

	sort.Strings(directoryNames)
	sort.Strings(fileNames)

	menuItems := make([]MenuItem, 0)
	for _, name := range directoryNames {
		menuItems = append(menuItems, MenuItem{label: name, color: config.FolderColor.Color})
	}
	for _, name := range fileNames {
		menuItems = append(menuItems, MenuItem{label: name, color: config.FileColor.Color})
	}

	return menuItems, nil
}

func (w *FileMenuWindow) run() (string, error) {
	gc.Cursor(0)
	defer gc.Cursor(1)
	currentPath := "."
	updateItems := true
	searchString := ""
	for {
		menuItems, err := w.getFiles(currentPath)
		if err != nil {
			return "", err
		}

		if updateItems {
			w.menuWindow.setItems(menuItems)
			updateItems = false
		}

		title := currentPath
		if searchString != "" {
			title = searchString
		}

		w.menuWindow.draw(title)

		ch := w.menuWindow.stdscr.GetChar() // TODO: dirt
		switch ch {
		case gc.KEY_ESC:
			if currentPath == "." {
				return "", nil
			}

			currentPath = filepath.Dir(currentPath)
			updateItems = true
		case gc.KEY_DOWN, gc.KEY_UP, gc.KEY_ENTER, gc.KEY_RETURN:
			selected := w.menuWindow.run(ch)
			if selected == "" {
				continue
			}

			info, err := os.Stat(filepath.Join(currentPath, selected))
			if err != nil {
				return "", err
			}

			currentPath = filepath.Join(currentPath, selected)
			if !info.IsDir() {
				return currentPath, nil
			}
			updateItems = true
		case gc.KEY_BACKSPACE:
			if searchString == "" {
				continue
			}

			searchString = searchString[:len(searchString)-1]

		default:
			chr := gc.KeyString(ch)
			if len(chr) > 1 {
				continue
			}

			searchString += chr
			updateItems = true
		}

		//e.draw()
		//selected, err := m.menuWindow.run(menuItems, currentPath)

	}
}
