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

func create(p string) {
    if err := os.MkdirAll(filepath.Dir(p), 0770); err != nil {
        return
    }
    os.Create(p)
}

func init() {
    tview.Borders.HorizontalFocus = tview.BoxDrawingsLightHorizontal
    tview.Borders.VerticalFocus = tview.BoxDrawingsLightVertical
    tview.Borders.TopLeftFocus = tview.BoxDrawingsLightDownAndRight
    tview.Borders.TopRightFocus = tview.BoxDrawingsLightDownAndLeft
    tview.Borders.BottomLeftFocus = tview.BoxDrawingsLightUpAndRight
    tview.Borders.BottomRightFocus = tview.BoxDrawingsLightUpAndLeft
}

func (v *velocity) updateList() {
    v.list.Clear()
    sort.Sort(byModTime(v.selectedFiles))
    for _, file := range v.selectedFiles {
        v.list.AddItem(file.path, "", 0, nil)
    }
    // TODO Set to getCurrentItem or 0
    v.list.SetCurrentItem(0)
}

func (v *velocity) editNote() *tcell.EventKey {

            var currentFile *file

            if v.list.GetItemCount() > 0 {
                currentFile = v.selectedFiles[v.list.GetCurrentItem()]
            } else {
                text := v.input.GetText()
                text = strings.TrimSpace(text)
                if text == "" {
                    return nil
                }
                path := text + ".txt"
                currentFile = &file{path: path, content: ""}
                v.allFiles = append(v.allFiles, currentFile)
                v.selectedFiles = append(v.selectedFiles, currentFile)
                create(path)
            }

            cmd := exec.Command(editor, currentFile.path)
            cmd.Stdout = os.Stdout
            cmd.Stdin = os.Stdin
            cmd.Stderr = os.Stderr

            v.app.Suspend(func() {
                cmd.Run()
            })
            content, err := ioutil.ReadFile(currentFile.path)
            if err != nil {
                panic(err)
            }
            currentFile.content = string(content)
            v.updateList()
            return nil
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

type velocity struct {
    selectedFiles []*file
    allFiles      []*file
    preview       *tview.TextView
    list          *tview.List
    input         *tview.InputField
    app           *tview.Application
}

func (v *velocity) scrollUp() *tcell.EventKey {
    _, _, _, height := v.preview.GetInnerRect()
    row, _ := v.preview.GetScrollOffset()
    v.preview.ScrollTo(row-height, 0)
    return nil
}

func (v *velocity) scrollDown() *tcell.EventKey {
    _, _, _, height := v.preview.GetInnerRect()
    row, _ := v.preview.GetScrollOffset()
    v.preview.ScrollTo(row+height, 0)
    return nil
}

func (v *velocity) run() {
    app := tview.NewApplication()

    v.app = app

    box := tview.NewFlex().SetDirection(tview.FlexRow)
    box.AddItem(v.input, 3, 1, true).
        AddItem(v.list, 0, 1, false).
        AddItem(v.preview, 0, 1, false)

    if err := app.SetRoot(box, true).Run(); err != nil {
        panic(err)
    }
}

func (v *velocity) listChanged (index int, _ string, _ string, _ rune) {
        if len(v.selectedFiles) == 0 {
            v.preview.SetText("")
            return
        }
        if index > len(v.selectedFiles)-1 {
            index = 0
        }
        v.preview.SetText(v.selectedFiles[index].content)
        v.preview.ScrollToBeginning()
        return
}

func (v *velocity) filterList (text string) {
        searchTerms := strings.Fields(text)
        v.selectedFiles = []*file{}
    FILES:
        for _, file := range v.allFiles {
            // search for search terms in content AND path
            content := strings.ToLower(file.content + " " + file.path)
            for _, term := range searchTerms {
                if !strings.Contains(content, strings.ToLower(term)) {
                    continue FILES
                }
            }
            v.selectedFiles = append(selectedFiles, file)
        }
        v.updateList()
    }

func getAllFiles(root string) []*file {
    var files = []*file{}
    err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if info.IsDir() {
            return nil
        }
        if !strings.HasSuffix(path, ".txt") {
            return nil
        }
        content, err := ioutil.ReadFile(path)
        if err != nil {
            panic(err)
        }
        files = append(files, &file{
            path:    path,
            content: string(content),
            modTime: info.ModTime(),
        })
        return nil
    })
    return files
}

func main() {

    v := velocity{}

    v.allFiles = getAllFiles(".")

    v.selectedFiles = allFiles

    v.input = tview.NewInputField()
    v.input.SetBorder(true)

    v.list = tview.NewList()
    v.list.SetBorder(true)
    v.list.ShowSecondaryText(false)
    v.list.SetHighlightFullLine(true)

    v.preview = tview.NewTextView()
    v.preview.SetBorder(true)

    v.list.SetChangedFunc( v.listChanged )
    v.input.SetChangedFunc(v.filterList)

    v.updateList()

    v.input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
        if event.Key() == tcell.KeyDown {
            v.list.SetCurrentItem(v.list.GetCurrentItem() + 1)
            return nil
        } else if event.Key() == tcell.KeyUp {
            current := v.list.GetCurrentItem()
            if current == 0 {
                return nil
            }
            v.list.SetCurrentItem(current - 1)
            return nil
        } else if event.Key() == tcell.KeyHome {
            return v.preview.ScrollToBeginning()
        } else if event.Key() == tcell.KeyEnd {
            return v.preview.ScrollToEnd()
        } else if event.Key() == tcell.KeyCtrlV {
            return v.scrollDown()
        } else if event.Key() == tcell.KeyCtrlB {
            return v.scrollUp()
        } else if event.Key() == tcell.KeyEnter {
            return v.editNote()
        }
        return event
    })
    v.run()
}
