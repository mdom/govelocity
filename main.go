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

var theme = tview.Theme{
    PrimitiveBackgroundColor:    tcell.ColorDefault,
    ContrastBackgroundColor:     tcell.ColorDefault,
    MoreContrastBackgroundColor: tcell.ColorDefault,
    BorderColor:                 tcell.ColorDefault,
    TitleColor:                  tcell.ColorDefault,
    GraphicsColor:               tcell.ColorDefault,
    PrimaryTextColor:            tcell.ColorDefault,
    SecondaryTextColor:          tcell.ColorDefault,
    TertiaryTextColor:           tcell.ColorDefault,
    InverseTextColor:            tcell.ColorDefault,
    ContrastSecondaryTextColor:  tcell.ColorDefault,
}

func init() {
    tview.Borders.HorizontalFocus = tview.BoxDrawingsLightHorizontal
    tview.Borders.VerticalFocus = tview.BoxDrawingsLightVertical
    tview.Borders.TopLeftFocus = tview.BoxDrawingsLightDownAndRight
    tview.Borders.TopRightFocus = tview.BoxDrawingsLightDownAndLeft
    tview.Borders.BottomLeftFocus = tview.BoxDrawingsLightUpAndRight
    tview.Borders.BottomRightFocus = tview.BoxDrawingsLightUpAndLeft
    tview.Styles = theme

}

type file struct {
    path    string
    content string
    modTime time.Time
}

type byModTime []*file

func (a byModTime) Len() int           { return len(a) }
func (a byModTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byModTime) Less(i, j int) bool { return a[i].modTime.Unix() > a[j].modTime.Unix() }

var editor = getEditor()

func create(p string) {
    if err := os.MkdirAll(filepath.Dir(p), 0770); err != nil {
        return
    }
    os.Create(p)
}

func (v *velocity) updateList() {
    index := v.list.GetCurrentItem()
    v.list.Clear()
    sort.Sort(byModTime(v.selectedFiles))
    for _, file := range v.selectedFiles {
        v.list.AddItem(strings.TrimSuffix(file.path, ".txt"), "", 0, nil)
    }
    v.list.SetCurrentItem(index)
}

func (v *velocity) editNote() {

    var currentFile *file

    if v.list.GetItemCount() > 0 {
        currentFile = v.selectedFiles[v.list.GetCurrentItem()]
    } else {
        text := v.input.GetText()
        text = strings.TrimSpace(text)
        if text == "" {
            return
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
    filenames     map[string]*file
}

func (v *velocity) scrollUp() {
    _, _, _, height := v.preview.GetInnerRect()
    row, _ := v.preview.GetScrollOffset()
    v.preview.ScrollTo(row-height, 0)
}

func (v *velocity) scrollDown() {
    _, _, _, height := v.preview.GetInnerRect()
    row, _ := v.preview.GetScrollOffset()
    v.preview.ScrollTo(row+height, 0)
}

func (v *velocity) run() {
    app := tview.NewApplication()
    v.app = app

    box := tview.NewFlex().SetDirection(tview.FlexRow)
    box.AddItem(v.input, 1, 1, true).
        AddItem(v.list, 0, 1, false).
        AddItem(v.preview, 0, 2, false)

    if err := app.SetRoot(box, true).Run(); err != nil {
        panic(err)
    }
}

func (v *velocity) listChanged(index int, _ string, _ string, _ rune) {
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

func (v *velocity) filterList(text string) {
    defer v.updateList()
    if text == "" {
        v.selectAllFiles()
        return
    }
    text = strings.ToLower(text)
    var newSelection []*file
    for _, i := range v.allFiles {
        if strings.Contains(strings.ToLower(i.path), text) ||
            strings.Contains(strings.ToLower(i.content), text) {
            newSelection = append(newSelection, i)
        }
    }
    v.selectedFiles = newSelection
}

func (v *velocity) getAllFiles(root string) {
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
        file := &file{
            path:    path,
            content: string(content),
            modTime: info.ModTime(),
        }
        files = append(files, file)
        v.filenames[path] = file
        return nil
    })
    if err != nil {
        panic(err)
    }
    v.allFiles = files

    v.selectAllFiles()
}

func (v *velocity) selectAllFiles() {
    if len(v.selectedFiles) == len(v.allFiles) {
        return
    }
    v.selectedFiles = make([]*file, len(v.allFiles))
    copy(v.selectedFiles, v.allFiles)
}

func newVelocity() *velocity {
    return &velocity{
        filenames: make(map[string]*file),
    }
}

func main() {

    v := newVelocity()

    dir := os.ExpandEnv("${HOME}/notes")

    if _, err := os.Stat(dir); os.IsNotExist(err) {
        err = os.Mkdir(dir, 0770)
        if err != nil {
            panic(err)
        }
    }

    err := os.Chdir(dir)
    if err != nil {
        panic(err)
    }

    v.getAllFiles(".")

    v.input = tview.NewInputField()
    v.input.SetLabel("> ")

    v.list = tview.NewList()
    v.list.ShowSecondaryText(false)
    v.list.SetHighlightFullLine(true)
    // v.list.SetSelectedReverseColor(true)

    v.preview = tview.NewTextView()
    v.preview.SetBorder(true)

    v.list.SetChangedFunc(v.listChanged)
    v.input.SetChangedFunc(v.filterList)

    v.updateList()

    var callbacks = map[tcell.Key]func(){
        tcell.KeyDown:   v.nextLine,
        tcell.KeyUp:     v.prevLine,
        tcell.KeyHome:   v.scrollToBeginning,
        tcell.KeyEnd:    v.scrollToEnd,
        tcell.KeyCtrlV:  v.scrollDown,
        tcell.KeyCtrlB:  v.scrollUp,
        tcell.KeyEnter:  v.editNote,
        tcell.KeyEscape: v.clearInput,
    }

    v.input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
        callback, ok := callbacks[event.Key()]
        if ok {
            callback()
            return nil
        }
        return event
    })
    v.run()
}

func (v *velocity) clearInput() {
    v.input.SetText("")
}

func (v *velocity) scrollToEnd() {
    v.preview.ScrollToBeginning()
}

func (v *velocity) scrollToBeginning() {
    v.preview.ScrollToBeginning()
}

func (v *velocity) prevLine() {
    current := v.list.GetCurrentItem()
    if current == 0 {
        return
    }
    v.list.SetCurrentItem(current - 1)
}

func (v *velocity) nextLine() {
    v.list.SetCurrentItem(v.list.GetCurrentItem() + 1)
}
