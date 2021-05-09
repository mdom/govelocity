package main

import (
    "github.com/gdamore/tcell/v2"
    "github.com/rivo/tview"
    "io/ioutil"
    "os"
    "os/exec"
    "path/filepath"
    "sort"
    "strings"
    "time"
)

type file struct {
    path    string
    content string
    modTime time.Time
}

type byModTime []*file

func (a byModTime) Len() int           { return len(a) }
func (a byModTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byModTime) Less(i, j int) bool { return a[i].modTime.Unix() > a[j].modTime.Unix() }

var allFiles = []*file{}
var selectedFiles []*file
var editor = getEditor()

func init() {
    tview.Borders.HorizontalFocus = tview.BoxDrawingsLightHorizontal
    tview.Borders.VerticalFocus = tview.BoxDrawingsLightVertical
    tview.Borders.TopLeftFocus = tview.BoxDrawingsLightDownAndRight
    tview.Borders.TopRightFocus = tview.BoxDrawingsLightDownAndLeft
    tview.Borders.BottomLeftFocus = tview.BoxDrawingsLightUpAndRight
    tview.Borders.BottomRightFocus = tview.BoxDrawingsLightUpAndLeft
}

func updateList(list *tview.List, files []*file) *tview.List {
    sort.Sort(byModTime(files))
    for _, file := range files {
        list.AddItem(file.path, "", 0, nil)
    }
    list.SetCurrentItem(0)
    return list
}

func getEditor() string {
    if e := os.Getenv("VISUAL"); e != "" {
        return e
    }
    if e := os.Getenv("EDITOR"); e != "" {
        return e
    }
    return "vi"
}

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
        allFiles = append(allFiles, &file{
            path:    path,
            content: string(content),
            modTime: info.ModTime(),
        })
        selectedFiles = allFiles
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

    list.SetChangedFunc(func(index int, _ string, _ string, _ rune) {
        if len(selectedFiles) == 0 {
            preview.SetText("")
            return
        }
        preview.SetText(selectedFiles[index].content)
        preview.ScrollToBeginning()
    })

    updateList(list, selectedFiles)

    input.SetChangedFunc(func(text string) {
        searchTerms := strings.Fields(text)
        selectedFiles = []*file{}
    FILES:
        for _, file := range allFiles {
            // search for search terms in content AND path
            content := strings.ToLower(file.content + " " + file.path)
            for _, term := range searchTerms {
                if !strings.Contains(content, strings.ToLower(term)) {
                    continue FILES
                }
            }
            selectedFiles = append(selectedFiles, file)
        }
        list.Clear()
        updateList(list, selectedFiles)
    })

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
        } else if event.Key() == tcell.KeyHome {
            preview.ScrollToBeginning()
        } else if event.Key() == tcell.KeyEnd {
            preview.ScrollToEnd()
        } else if event.Key() == tcell.KeyCtrlV {
            _, _, _, height := preview.GetInnerRect()
            row, _ := preview.GetScrollOffset()
            preview.ScrollTo(row+height, 0)
        } else if event.Key() == tcell.KeyCtrlB {
            _, _, _, height := preview.GetInnerRect()
            row, _ := preview.GetScrollOffset()
            preview.ScrollTo(row-height, 0)
        } else if event.Key() == tcell.KeyEnter {

            var currentFile *file

            if list.GetItemCount() > 0 {
                currentFile = selectedFiles[list.GetCurrentItem()]
            } else {
                text := input.GetText()
                text = strings.TrimSpace(text)
                if text == "" {
                    return nil
                }
                path := text + ".txt"
                currentFile = &file{path: path, content: ""}
                allFiles = append(allFiles, currentFile)
            }

            cmd := exec.Command(editor, currentFile.path)
            cmd.Stdout = os.Stdout
            cmd.Stdin = os.Stdin
            cmd.Stderr = os.Stderr

            app.Suspend(func() {
                cmd.Run()
            })
            content, err := ioutil.ReadFile(currentFile.path)
            if err != nil {
                panic(err)
            }
            currentFile.content = string(content)
            preview.SetText(currentFile.content)
            return nil
        }
        return event
    })

    box.AddItem(input, 3, 1, true).
        AddItem(list, 0, 1, false).
        AddItem(preview, 0, 1, false)

    if err := app.SetRoot(box, true).Run(); err != nil {
        panic(err)
    }
}
