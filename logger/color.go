package logger

import (
	"image/color"

	"github.com/muesli/termenv"
)

// catppuccin colors https://catppuccin.com/palette/
//
//nolint:gochecknoglobals // colors are global constants
var (
	LattePeach    = color.RGBA{254, 100, 11, 255}
	LatteRed      = color.RGBA{210, 15, 57, 255}
	LatteSky      = color.RGBA{4, 165, 229, 255}
	LatteSubtext0 = color.RGBA{108, 111, 133, 255}
	LatteSubtext1 = color.RGBA{92, 95, 119, 255}
	LatteText     = color.RGBA{76, 79, 105, 255}
	MochaPeach    = color.RGBA{250, 179, 135, 255}
	MochaRed      = color.RGBA{243, 139, 168, 255}
	MochaSky      = color.RGBA{137, 220, 235, 255}
	MochaSubtext0 = color.RGBA{166, 173, 200, 255}
	MochaSubtext1 = color.RGBA{186, 194, 222, 255}
	MochaText     = color.RGBA{205, 214, 244, 255}
)

type Colors struct {
	Peach      termenv.Color
	Red        termenv.Color
	Sky        termenv.Color
	Subsubtext termenv.Color
	Subtext    termenv.Color
	Text       termenv.Color
}

func colors(output *termenv.Output) Colors {
	colors := Colors{}

	if output.EnvNoColor() { //nolint:gocritic // if-else is better here
		colors.Peach = termenv.NoColor{}
		colors.Red = termenv.NoColor{}
		colors.Sky = termenv.NoColor{}
		colors.Subsubtext = termenv.NoColor{}
		colors.Subtext = termenv.NoColor{}
		colors.Text = termenv.NoColor{}
	} else if output.HasDarkBackground() {
		colors.Peach = output.FromColor(MochaPeach)
		colors.Red = output.FromColor(MochaRed)
		colors.Sky = output.FromColor(MochaSky)
		colors.Subsubtext = output.FromColor(MochaSubtext0)
		colors.Subtext = output.FromColor(MochaSubtext1)
		colors.Text = output.FromColor(MochaText)
	} else {
		colors.Peach = output.FromColor(LattePeach)
		colors.Red = output.FromColor(LatteRed)
		colors.Sky = output.FromColor(LatteSky)
		colors.Subsubtext = output.FromColor(LatteSubtext0)
		colors.Subtext = output.FromColor(LatteSubtext1)
		colors.Text = output.FromColor(LatteText)
	}

	return colors
}
