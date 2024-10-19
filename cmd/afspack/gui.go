package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/anasrar/afs/internal/metadata"
	rayguistyle "github.com/anasrar/afs/internal/raygui_style"
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

func drop(filePath string) error {
	metadataBuf, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var m metadata.Metadata
	if err := json.Unmarshal(metadataBuf, &m); err != nil {
		return err
	}

	writeLog(fmt.Sprintf("AFS Version: %X", m.Version))
	writeLog(fmt.Sprintf("AFS Attributes Info: %d", m.AttributesInfo))
	writeLog(fmt.Sprintf("AFS Entry Block Alignment: %d", m.EntryBlockAlignment))
	writeLog(fmt.Sprintf("AFS Entry Total: %d", m.EntryTotal))
	writeLog("Ready")

	return nil
}

func gui() {
	rl.InitWindow(int32(width), int32(height), "AFS Packer")
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

			if err := drop(filePath); err != nil {
				writeLog(err.Error())
				metadataPath = ""
			} else {
				metadataPath = filePath
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

			if packing || metadataPath == "" {
				raygui.Disable()
			}

			if raygui.Button(rl.NewRectangle(width-180, height-40, 82, 32), "Pack") {
				ctx, cancel = context.WithCancel(context.Background())

				go func() {
					if err := pack(
						ctx,
						metadataPath,
						func(total, current uint32, name string) {
							writeLog(fmt.Sprintf("%d/%d(%s): start", current, total, name))
						},
						func(total, current uint32, name string) {
							writeLog(fmt.Sprintf("%d/%d(%s): done", current, total, name))

							progress = float32(current) / float32(total)
							if total == current {
								writeLog("Done")
								packing = false
								progress = 0
							}
						},
					); err != nil {
						writeLog(err.Error())

						packing = false
						progress = 0
					}
				}()

				packing = true
				progress = 0
			}

			if packing || metadataPath == "" {
				raygui.Enable()
			}

			if !packing {
				raygui.Disable()
			}

			if raygui.Button(rl.NewRectangle(width-90, height-40, 82, 32), "Cancel") {
				cancel()
				writeLog("Plase Wait For Cancellation")
			}

			if !packing {
				raygui.Enable()
			}
		}

		rl.EndDrawing()
	}

	rayguistyle.Unload()

	rl.CloseWindow()
}
