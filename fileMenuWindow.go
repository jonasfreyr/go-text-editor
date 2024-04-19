package main

import (
	"github.com/lithammer/fuzzysearch/fuzzy"
	gc "github.com/rthornton128/goncurses"
	"github.com/yireyun/go-queue"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type FileMenuWindow struct {
	menuWindow *MenuWindow

	subFiles map[string][]string
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
		menuItems = append(menuItems, MenuItem{label: name, value: filepath.Join(currentPath, name), color: config.FolderColor.Color})
	}
	for _, name := range fileNames {
		menuItems = append(menuItems, MenuItem{label: name, value: filepath.Join(currentPath, name), color: config.FileColor.Color})
	}

	return menuItems, nil
}

func (w *FileMenuWindow) updateAllSubFilesAndDirectories(path string) error {
	before := time.Now()

	w.subFiles = make(map[string][]string)

	q := queue.NewQueue(1000)
	q.Put(path)

	for q.Quantity() > 0 {
		c, _, _ := q.Get()

		currentPath, _ := c.(string)
		w.subFiles[currentPath] = make([]string, 0)

		fi, err := os.ReadDir(currentPath)
		if err != nil {
			return err
		}
		for _, subFile := range fi {
			name := filepath.Join(currentPath, subFile.Name())

			// log.Println("you got mail:", name)

			if subFile.IsDir() {
				q.Put(name)
				continue
			}
			w.subFiles[currentPath] = append(w.subFiles[currentPath], name)
		}
	}

	log.Println("walk: ", time.Since(before))
	return nil
}

func (w *FileMenuWindow) getAllSublists(path string) []string {
	res := make([]string, 0)
	for key, vals := range w.subFiles {
		if strings.Contains(key, path) || path == "." {
			res = append(res, vals...)
		}
	}
	return res
}

func (w *FileMenuWindow) fuzzyFind(searchString, path string) ([]MenuItem, error) {
	subFiles := w.getAllSublists(path)
	res := fuzzy.Find(searchString, subFiles)

	sort.Slice(res, func(i, j int) bool {
		return len(res[i]) < len(res[j])
	})

	config := GetEditorConfig()

	menuItems := make([]MenuItem, len(res))
	for i, path := range res {
		menuItems[i] = MenuItem{
			label: path,
			color: config.FileColor.Color,
			value: path,
		}
	}

	return menuItems, nil
}

func (w *FileMenuWindow) run() (string, error) {
	gc.Cursor(0)
	defer gc.Cursor(1)
	currentPath := "."
	updateItems := true
	searchString := ""
	err := w.updateAllSubFilesAndDirectories(currentPath)
	if err != nil {
		return "", err
	}
	for {
		var menuItems []MenuItem
		if searchString == "" {
			menuItems, err = w.getFiles(currentPath)
		} else {
			menuItems, err = w.fuzzyFind(searchString, currentPath)
		}

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

			info, err := os.Stat(selected)
			if err != nil {
				return "", err
			}

			currentPath = selected
			if !info.IsDir() {
				return currentPath, nil
			}
			updateItems = true
		case gc.KEY_BACKSPACE:
			if searchString == "" {
				continue
			}

			searchString = searchString[:len(searchString)-1]
			updateItems = true
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
