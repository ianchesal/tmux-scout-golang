package main

func runPicker(statusFilePath, currentPane string) {
	scoutDir := defaultScoutDir()
	result := Sync(statusFilePath, scoutDir)
	Render(result.Status, currentPane, result.Panes)
}

func runPickerPreview(paneID, statusFile string) {
	PreviewPane(paneID, statusFile)
}
