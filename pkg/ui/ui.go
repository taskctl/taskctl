package ui
//
//import (
//	ui "github.com/gizak/termui/v3"
//	"github.com/gizak/termui/v3/widgets"
//	"log"
//)
//
//var grid *ui.Grid
//var p *widgets.Paragraph
//var tasksui *TaskUI
//
//type TaskOutput struct {
//	p *widgets.Paragraph
//	out chan string
//}
//
//func (to *TaskOutput) Listen() {
//	for {
//		select {
//		case s := <-to.out:
//			p.Text = p.Text + s
//			ui.Render(grid)
//		}
//	}
//}
//
//type TaskUI struct {
//	output chan string
//	done chan bool
//}
//
//func (taskUI *TaskUI) Close() {
//	ui.Close()
//}
//
//func NewTaskUi(done chan bool, tasks []*Task) {
//	if err := ui.Init(); err != nil {
//		log.Fatalf("failed to initialize termui: %v", err)
//	}
//	defer ui.Close()
//
//	tasksui = &TaskUI{
//		done: done,
//	}
//
//	grid = ui.NewGrid()
//	termWidth, termHeight := ui.TerminalDimensions()
//	grid.SetRect(0, 0, termWidth, termHeight)
//
//	ls := widgets.NewList()
//
//	p = widgets.NewParagraph()
//	p.Text = ""
//	p.Title = "Task output"
//
//	grid.Set(
//		ui.NewRow(1,
//			ui.NewCol(0.75, p),
//			ui.NewCol(0.25, ls),
//		),
//	)
//
//	ui.Render(grid)
//
//	uiEvents := ui.PollEvents()
//
//	for {
//		var rows = []string{}
//		for _, t := range tasks {
//			var r string
//			switch t.readStatus() {
//			case STATUS_DONE:
//				r += "[✓]"
//			case STATUS_RUNNING:
//				r += "[⧗]"
//			default:
//				r += "[-]"
//			}
//
//			r += t.Name
//			rows = append(rows, r)
//		}
//
//		ls.Rows = rows
//
//		select {
//		case e := <-uiEvents:
//			switch e.ID {
//			case "q", "<C-c>":
//				return
//			}
//		case <-tasksui.done:
//			return
//		default:
//			ui.Render()
//		}
//	}
//}
//
//func NewTaskOutput(out chan string) *TaskOutput{
//	return &TaskOutput{
//		p:   p,
//		out: out,
//	}
//}
//
