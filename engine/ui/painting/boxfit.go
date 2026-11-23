package painting

import "hexago/ui/geom"

// BoxFit — как картинка вписывается в прямоугольник, как BoxFit во Flutter.
type BoxFit uint8

const (
	// BoxFitScaleDown — как есть, а если не помещается — уменьшить целиком (как Contain).
	// Это значение по умолчанию.
	BoxFitScaleDown BoxFit = iota
	BoxFitContain          // целиком, как можно крупнее; по краям могут остаться поля
	BoxFitCover            // заполнить прямоугольник; лишнее по краям обрезается
	BoxFitFill             // растянуть на весь прямоугольник, не сохраняя пропорции
	BoxFitWidth            // во всю ширину; по высоте — обрезать или оставить поля
	BoxFitHeight           // во всю высоту; по ширине — обрезать или оставить поля
	BoxFitNone             // как есть, без масштаба; что не помещается — обрезается
)

// FitSizes — какую часть картинки размера input показать (source, в размерах картинки) и в каком
// размере (dest) в прямоугольнике размера box. Где именно они стоят, решает выравнивание (Inscribe).
func FitSizes(fit BoxFit, input, box geom.Size) (source, dest geom.Size) {
	if input.IsEmpty() || box.IsEmpty() {
		return geom.Size{}, geom.Size{}
	}
	wider := box.Width/box.Height > input.Width/input.Height // прямоугольник «шире» картинки
	switch fit {
	case BoxFitFill:
		return input, box
	case BoxFitContain:
		if wider {
			return input, geom.Size{Width: input.Width * box.Height / input.Height, Height: box.Height}
		}
		return input, geom.Size{Width: box.Width, Height: input.Height * box.Width / input.Width}
	case BoxFitCover:
		if wider {
			return geom.Size{Width: input.Width, Height: input.Width * box.Height / box.Width}, box
		}
		return geom.Size{Width: input.Height * box.Width / box.Height, Height: input.Height}, box
	case BoxFitWidth:
		if wider {
			return geom.Size{Width: input.Width, Height: input.Width * box.Height / box.Width}, box
		}
		return input, geom.Size{Width: box.Width, Height: input.Height * box.Width / input.Width}
	case BoxFitHeight:
		if wider {
			return input, geom.Size{Width: input.Width * box.Height / input.Height, Height: box.Height}
		}
		return geom.Size{Width: input.Height * box.Width / box.Height, Height: input.Height}, box
	case BoxFitNone:
		s := geom.Size{Width: min(input.Width, box.Width), Height: min(input.Height, box.Height)}
		return s, s
	}
	// BoxFitScaleDown
	dest = input
	if dest.Height > box.Height {
		dest = geom.Size{Width: box.Height * input.Width / input.Height, Height: box.Height}
	}
	if dest.Width > box.Width {
		dest = geom.Size{Width: box.Width, Height: box.Width * input.Height / input.Width}
	}
	return input, dest
}

// Inscribe ставит прямоугольник размера s внутрь r по выравниванию a.
func Inscribe(a geom.Alignment, s geom.Size, r geom.Rect) geom.Rect {
	off := a.Along(geom.Size{Width: r.Width() - s.Width, Height: r.Height() - s.Height})
	return geom.RectXYWH(r.Left+off.X, r.Top+off.Y, s.Width, s.Height)
}
