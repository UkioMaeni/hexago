package icons_test

import (
	"testing"

	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/hexagotest"
	"hexago/ui/icons"
	"hexago/ui/widgets"
)

// all — все значки Material из пакета, по порядку объявления.
var all = []*icons.Data{
	icons.Add, icons.AddCircle, icons.Remove, icons.Close, icons.Check, icons.CheckCircle, icons.DoneAll,
	icons.Menu, icons.ArrowForward, icons.ArrowBack, icons.MoreVert, icons.MoreHoriz, icons.ChevronLeft,
	icons.ChevronRight, icons.ExpandMore, icons.ExpandLess, icons.Search, icons.Home, icons.Favorite,
	icons.FavoriteBorder, icons.Star, icons.StarBorder, icons.Settings, icons.Person, icons.AccountCircle,
	icons.Delete, icons.Edit, icons.Share, icons.Info, icons.Warning, icons.Error, icons.Notifications,
	icons.Mail, icons.Phone, icons.PhotoCamera, icons.Image, icons.Send, icons.Refresh, icons.Download,
	icons.Upload, icons.Visibility, icons.VisibilityOff, icons.Lock, icons.PlayArrow, icons.Pause,
	icons.ShoppingCart, icons.Place, icons.CalendarToday, icons.List, icons.GridView,
}

func TestAllIconsParseInsideBox(t *testing.T) {
	for i, ic := range all {
		b := ic.Path().Bounds()
		if b.Left < -0.5 || b.Top < -0.5 || b.Right > 24.5 || b.Bottom > 24.5 || b.Width() < 4 {
			t.Errorf("значок %d: границы %v — не в квадрате 24×24", i, b)
		}
	}
}

// TestIconsGolden — все значки сеткой по 10 в ряд, 32 dp, при плотности 2.
func TestIconsGolden(t *testing.T) {
	var rows []widgets.Widget
	for i := 0; i < len(all); i += 10 {
		var row []widgets.Widget
		for _, ic := range all[i:min(i+10, len(all))] {
			row = append(row, widgets.Padding{Padding: geom.All(4), Child: widgets.Icon{Icon: ic, Size: 32}})
		}
		rows = append(rows, widgets.Row{Children: row})
	}
	tree := widgets.IconTheme{
		Data:  widgets.IconThemeData{Color: gfx.ColorFromHex("#1D1B20")},
		Child: widgets.Padding{Padding: geom.All(4), Child: widgets.Column{Children: rows}},
	}
	hexagotest.Golden(t, "icons", hexagotest.Render(tree, 416*2, 208*2, 2))
}
