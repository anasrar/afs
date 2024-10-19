package main

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	rayguistyle "github.com/anasrar/afs/internal/raygui_style"
	"github.com/anasrar/afs/pkg/afs"
	"github.com/gen2brain/raylib-go/raygui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func writeLog(msg string) {
	result := ""
	words := strings.Split(msg, " ")
	w := float32(0)

	for _, word := range words {
		dimension := rl.MeasureTextEx(rayguistyle.DefaultFont, word, 14, 0)
		w += dimension.X
		if w >= (logContentRectangle.Width - 22) { // 22 is padding
			result += "\n"
			w = 0
		}
		result += word + " "
	}

	logs += result

	{
		dimension := rl.MeasureTextEx(rayguistyle.DefaultFont, logs, 16, 0)
		logContentRectangle.Height = dimension.Y + 16 // 16 is padding

		if logAutoScroll && dimension.Y > logRectangle.Height {
			offset := logContentRectangle.Height - logRectangle.Height
			logScroll.Y = -offset
		}
	}

	logs += "\n"
}

func clearLog() {
	logContentRectangle = rl.NewRectangle(0, 0, 234, 0)
	logScroll = rl.NewVector2(0, 0)
	logs = ""
}

func gui() {
	rl.InitWindow(int32(width), int32(height), "AFS Unpacker")
	rl.SetTargetFPS(30)

	rayguistyle.Load()

	for !rl.WindowShouldClose() {

		if rl.IsWindowResized() {
			width = float32(rl.GetScreenWidth())
			height = float32(rl.GetScreenHeight())

			logRectangle = rl.NewRectangle(0, 0, width, height-48)
			logContentRectangle.Width = width - 20
		}

		if rl.IsFileDropped() {
			filePath := rl.LoadDroppedFiles()[0]

			a := afs.New()
			if err := afs.FromPath(a, filePath); err != nil {
				writeLog(err.Error())

				afsPath = ""
			} else {
				writeLog(fmt.Sprintf("AFS Version: %X", a.Version))
				writeLog(fmt.Sprintf("AFS Attributes Info: %d", a.AttributesInfo))
				writeLog(fmt.Sprintf("AFS Entry Block Alignment: %d", a.EntryBlockAlignment))
				writeLog(fmt.Sprintf("AFS Entry Total: %d", a.EntryTotal))
				writeLog("Ready")

				afsPath = filePath
			}

			rl.UnloadDroppedFiles()
		}

		rl.BeginDrawing()
		rl.ClearBackground(rl.NewColor(0x12, 0x12, 0x12, 0xFF))

		{
			raygui.ScrollPanel(
				logRectangle,
				"",
				logContentRectangle,
				&logScroll,
				&logView,
			)

			// rl.DrawRectangle(
			// 	int32(logRectangle.X+logScroll.X),
			// 	int32(logRectangle.Y+logScroll.Y),
			// 	int32(logContentRectangle.Width),
			// 	int32(logContentRectangle.Height),
			// 	rl.Fade(rl.Red, 0.1),
			// )

			rl.BeginScissorMode(
				int32(logView.X),
				int32(logView.Y),
				int32(logView.Width),
				int32(logView.Height),
			)

			rl.DrawTextEx(
				rayguistyle.DefaultFont,
				logs,
				rl.NewVector2(
					logRectangle.X+logScroll.X+8,
					logRectangle.Y+logScroll.Y+8,
				),
				16,
				0,
				rl.NewColor(0xDA, 0xDA, 0xDA, 0xFF),
			)

			rl.EndScissorMode()

			raygui.ProgressBar(rl.NewRectangle(214, height-40, width-402, 32), "", "", progress, 0.0, 1.0)

			if raygui.Button(rl.NewRectangle(8, height-40, 82, 32), "Clear") {
				clearLog()
			}

			logAutoScroll = raygui.CheckBox(rl.NewRectangle(98, height-30, 12, 12), "Auto Scroll", logAutoScroll)

			if unpacking || afsPath == "" {
				raygui.Disable()
			}

			if raygui.Button(rl.NewRectangle(width-180, height-40, 82, 32), "Unpack") {
				ctx, cancel = context.WithCancel(context.Background())

				go func() {
					if err := unpack(
						ctx,
						afsPath,
						func(total, current uint32, name string) {
							writeLog(fmt.Sprintf("%d/%d(%s): start", current, total, name))
						},
						func(total, current uint32, name string) {
							writeLog(fmt.Sprintf("%d/%d(%s): done", current, total, name))

							progress = float32(current) / float32(total)
							if total == current {
								writeLog("Done")
								unpacking = false
								progress = 0
							}
						},
					); err != nil {
						writeLog(err.Error())

						unpacking = false
						progress = 0
					}
				}()

				unpacking = true
				progress = 0
			}

			if unpacking || afsPath == "" {
				raygui.Enable()
			}

			if !unpacking {
				raygui.Disable()
			}

			if raygui.Button(rl.NewRectangle(width-90, height-40, 82, 32), "Cancel") {
				cancel()
				writeLog("Plase Wait For Cancellation")
			}

			if !unpacking {
				raygui.Enable()
			}
		}

		rl.EndDrawing()
	}

	rayguistyle.Unload()

	rl.CloseWindow()
}
