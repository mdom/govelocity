package main

import (
    "github.com/gdamore/tcell/v2"
    "github.com/rivo/tview"
    "io/ioutil"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
)

type file struct {
    path    string
    content string
}

var allFiles = make(map[string]*file)
var selectedFiles = allFiles

func main() {
    tview.Styles.InverseTextColor = tcell.ColorWhite

    app := tview.NewApplication()

    root := "."
    err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if info.IsDir() {
            return nil
        }
        if !strings.HasSuffix(path, ".txt") {
            return nil
        }
        content, err := ioutil.ReadFile(path)
        if err != nil {
            return nil
        }
        allFiles[path] = &file{path: path, content: string(content)}
        return nil
    })

    if err != nil {
        panic(err)
    }

    box := tview.NewFlex().SetDirection(tview.FlexRow)
    input := tview.NewInputField()
    input.SetBorder(true)

    list := tview.NewList()
    list.SetBorder(true)
    list.ShowSecondaryText(false)
    list.SetHighlightFullLine(true)

    preview := tview.NewTextView()
    preview.SetBorder(true)

    input.SetChangedFunc(func(text string) {
        searchTerms := strings.Fields(text)
        selectedFiles = make(map[string]*file)
    FILES:
        for _, file := range allFiles {
            // search for search terms in content and path
            content := file.content + " " + file.path
            for _, term := range searchTerms {
                if !strings.Contains(content, term) {
                    continue FILES
                }
            }
            selectedFiles[file.path] = file
        }
        list.Clear()
        for _, file := range selectedFiles {
            list.AddItem(file.path, "", 0, nil)
            list.SetCurrentItem(0)
        }
    })

    list.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
        preview.SetText(allFiles[mainText].content)
        preview.ScrollToBeginning()
    })

    for _, file := range allFiles {
        list.AddItem(file.path, "", 0, nil)
    }

    list.SetCurrentItem(0)

    box.AddItem(input, 3, 1, true).
        AddItem(list, 0, 1, false).
        AddItem(preview, 0, 1, false)

    input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
        if event.Key() == tcell.KeyDown {
            list.SetCurrentItem(list.GetCurrentItem() + 1)
            return nil
        } else if event.Key() == tcell.KeyUp {
            current := list.GetCurrentItem()
            if current == 0 {
                return nil
            }
            list.SetCurrentItem(current - 1)
            return nil
        } else if event.Key() == tcell.KeyEnter {

            var path string

            if list.GetItemCount() > 0 {
                path, _ = list.GetItemText(list.GetCurrentItem())
            } else {
                text := input.GetText()
                text = strings.TrimSpace(text)
                if text == "" {
                    return nil
                }
                path = text + ".txt"
                allFiles[path] = &file{path: path, content: ""}
            }

            cmd := exec.Command("vi", path)
            cmd.Stdout = os.Stdout
            cmd.Stdin = os.Stdin
            cmd.Stderr = os.Stderr

            app.Suspend(func() {
                cmd.Run()
            })
            content, err := ioutil.ReadFile(path)
            if err != nil {
                panic(err)
            }
            allFiles[path].content = string(content)
            return nil
        }
        return event
    })

    if err := app.SetRoot(box, true).Run(); err != nil {
        panic(err)
    }
}
