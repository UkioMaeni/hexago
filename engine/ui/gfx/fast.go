package gfx

import (
	"image"
	"math"

	"hexago/ui/geom"
)

// Быстрые пути холста. Прямоугольники, скруглённые прямоугольники и круги без поворота — это почти
// всё, из чего состоит интерфейс (фоны, кнопки, карточки, ползунки). Их покрытие считается прямо по
// геометрии: внутри фигуры пиксели заливаются целиком, сглаживаются только края и углы (по расстоянию
// до дуги). Растеризатор контуров для них не нужен.

// div255 — x/255 с округлением, без деления.
func div255(x uint32) uint32 {
	x += 128
	return (x + x>>8) >> 8
}

// mix накладывает цвет (r, g, b) с непрозрачностью a (0…255) поверх пикселя dst (0xAABBGGRR).
func mix(dst, r, g, b, a uint32) uint32 {
	if a >= 255 {
		return 0xFF000000 | b<<16 | g<<8 | r
	}
	return mix2(dst, (b<<16|r)*a+0x00800080, (0xFF<<16|g)*a+0x00800080, 255-a)
}

// mix2 — смешивание по два канала за раз: R и B лежат в одном числе (0x00BB00RR), G и A — в другом,
// каждый в своих 16 битах. srcRB и srcGA — каналы цвета, уже умноженные на непрозрачность, плюс
// 0x80 на округление; inv = 255 − непрозрачность. Результат тот же, что у div255 по каждому каналу.
func mix2(dst, srcRB, srcGA, inv uint32) uint32 {
	rb := (dst&0x00FF00FF)*inv + srcRB
	rb = (rb + (rb>>8)&0x00FF00FF) >> 8 & 0x00FF00FF
	ga := (dst>>8&0x00FF00FF)*inv + srcGA
	ga = (ga + (ga>>8)&0x00FF00FF) >> 8 & 0x00FF00FF
	return rb | ga<<8
}

// fillRRectFast заливает скруглённый прямоугольник сплошным цветом без растеризатора.
// Возвращает false, если быстрый путь не подходит (поворот, отражение) — тогда рисует общий путь.
func (c *Canvas) fillRRectFast(rr geom.RRect, col Color) bool {
	m := c.st.m
	if m.B != 0 || m.C != 0 || m.A <= 0 || m.D <= 0 {
		return false
	}
	ca := uint32(float64(col.A)*c.st.alpha + 0.5)
	if ca == 0 {
		return true
	}
	rr = rr.Clamped()
	x0, x1 := m.A*rr.Left+m.E, m.A*rr.Right+m.E
	y0, y1 := m.D*rr.Top+m.F, m.D*rr.Bottom+m.F
	if x1 <= x0 || y1 <= y0 {
		return true
	}
	scale := func(r geom.Radius) geom.Radius { return geom.Radius{X: r.X * m.A, Y: r.Y * m.D} }
	tl, tr := scale(rr.Radius.TopLeft), scale(rr.Radius.TopRight)
	br, bl := scale(rr.Radius.BottomRight), scale(rr.Radius.BottomLeft)

	b := image.Rect(int(math.Floor(x0)), int(math.Floor(y0)), int(math.Ceil(x1)), int(math.Ceil(y1))).Intersect(c.st.clip)
	if b.Empty() {
		return true
	}
	r, g, bl8 := uint32(col.R), uint32(col.G), uint32(col.B)
	mask := c.st.clipMask
	for y := b.Min.Y; y < b.Max.Y; y++ {
		fy := float64(y)
		cy := overlap(fy, fy+1, y0, y1)
		py := fy + 0.5
		// Какие углы задевает эта строка: слева — верхний левый или нижний левый, справа — так же.
		var left, right geom.Radius
		var lcy, rcy float64 // центр угла по y
		switch {
		case py < y0+tl.Y:
			left, lcy = tl, y0+tl.Y
		case py > y1-bl.Y:
			left, lcy = bl, y1-bl.Y
		}
		switch {
		case py < y0+tr.Y:
			right, rcy = tr, y0+tr.Y
		case py > y1-br.Y:
			right, rcy = br, y1-br.Y
		}
		// Пиксели [inL, inR) лежат целиком между углами и краями: покрытие — только по вертикали.
		inL := int(math.Ceil(x0 + left.X))
		inR := int(math.Floor(x1 - right.X))
		row := c.pix[y*c.stride:]
		maskRow := 0
		if mask != nil {
			maskRow = (y-mask.Rect.Min.Y)*mask.Stride - mask.Rect.Min.X
		}
		edge := func(x int) {
			fx := float64(x)
			cov := overlap(fx, fx+1, x0, x1) * cy
			px := fx + 0.5
			if left.X > 0 && px < x0+left.X {
				cov = math.Min(cov, ellipseCover(px, py, x0+left.X, lcy, left))
			} else if right.X > 0 && px > x1-right.X {
				cov = math.Min(cov, ellipseCover(px, py, x1-right.X, rcy, right))
			}
			k := uint32(cov*255 + 0.5)
			if mask != nil {
				k = div255(k * uint32(mask.Pix[maskRow+x]))
			}
			if k > 0 {
				row[x] = mix(row[x], r, g, bl8, div255(k*ca))
			}
		}
		// Края и углы — по пикселю, середина строки — сплошным отрезком.
		spanL, spanR := max(inL, b.Min.X), min(inR, b.Max.X)
		if spanR <= spanL {
			for x := b.Min.X; x < b.Max.X; x++ {
				edge(x)
			}
			continue
		}
		for x := b.Min.X; x < spanL; x++ {
			edge(x)
		}
		k := uint32(cy*255 + 0.5)
		if mask == nil {
			fillSpan(row[spanL:spanR], r, g, bl8, div255(k*ca))
		} else {
			for x := spanL; x < spanR; x++ {
				if km := div255(k * uint32(mask.Pix[maskRow+x])); km > 0 {
					row[x] = mix(row[x], r, g, bl8, div255(km*ca))
				}
			}
		}
		for x := spanR; x < b.Max.X; x++ {
			edge(x)
		}
	}
	return true
}

// fillSpan накладывает цвет с непрозрачностью a на отрезок строки: непрозрачный — просто записывает.
func fillSpan(span []uint32, r, g, b, a uint32) {
	if a == 0 || len(span) == 0 {
		return
	}
	if a >= 255 {
		p := 0xFF000000 | b<<16 | g<<8 | r
		span[0] = p
		for n := 1; n < len(span); n *= 2 { // копированием удвоением — быстрее цикла
			copy(span[n:], span[:n])
		}
		return
	}
	srcRB, srcGA, inv := (b<<16|r)*a+0x00800080, (0xFF<<16|g)*a+0x00800080, 255-a
	for i, d := range span {
		span[i] = mix2(d, srcRB, srcGA, inv)
	}
}

// overlap — какая доля отрезка [a, a+1) пикселя попадает в [lo, hi].
func overlap(a, b, lo, hi float64) float64 {
	return math.Max(0, math.Min(b, hi)-math.Max(a, lo))
}

// ellipseCover — доля пикселя с центром (px, py) внутри эллипса с центром (cx, cy) и радиусами r:
// по расстоянию до края эллипса (сглаживание шириной в пиксель).
func ellipseCover(px, py, cx, cy float64, r geom.Radius) float64 {
	if r.X <= 0 || r.Y <= 0 {
		return 1
	}
	dx, dy := (px-cx)/r.X, (py-cy)/r.Y
	f := dx*dx + dy*dy - 1
	// |∇f| в пикселях: f — в долях радиуса, поэтому делим на радиусы.
	gx, gy := 2*dx/r.X, 2*dy/r.Y
	grad := math.Sqrt(gx*gx + gy*gy)
	if grad == 0 {
		return 1
	}
	return math.Max(0, math.Min(1, 0.5-f/grad))
}

// DrawLayer переносит на холст готовую картинку слоя layer — другого холста, на котором рисовали
// с прозрачного фона (его цвета уже умножены на свою непрозрачность). Левый верхний пиксель слоя
// ложится в точку at (пиксели холста, преобразование не применяется). Учитываются обрезка и общая
// прозрачность холста. opaque — в слое нет полупрозрачных пикселей: строки просто копируются.
//
// Так экран во время перехода рисуется один раз, а в кадрах только переносится со сдвигом.
func (c *Canvas) DrawLayer(layer *Canvas, at image.Point, opaque bool) {
	r := image.Rect(at.X, at.Y, at.X+layer.width, at.Y+layer.height).Intersect(c.st.clip)
	a := uint32(c.st.alpha*255 + 0.5)
	if r.Empty() || a == 0 {
		return
	}
	mask := c.st.clipMask
	for y := r.Min.Y; y < r.Max.Y; y++ {
		dst := c.pix[y*c.stride+r.Min.X : y*c.stride+r.Max.X]
		src := layer.pix[(y-at.Y)*layer.stride+r.Min.X-at.X:]
		src = src[:len(dst)]
		if opaque && a == 255 && mask == nil {
			copy(dst, src)
			continue
		}
		maskRow := 0
		if mask != nil {
			maskRow = (y-mask.Rect.Min.Y)*mask.Stride + r.Min.X - mask.Rect.Min.X
		}
		for i, p := range src {
			k := a
			if mask != nil {
				k = div255(k * uint32(mask.Pix[maskRow+i]))
			}
			if k == 0 || p>>24 == 0 {
				continue
			}
			if k == 255 && p>>24 == 255 {
				dst[i] = p
				continue
			}
			// Пиксель слоя с непрозрачностью k: q = p·k; результат = q + dst·(1 − q.A).
			qrb := (p&0x00FF00FF)*k + 0x00800080
			qrb = (qrb + (qrb>>8)&0x00FF00FF) >> 8 & 0x00FF00FF
			qga := (p>>8&0x00FF00FF)*k + 0x00800080
			qga = (qga + (qga>>8)&0x00FF00FF) >> 8 & 0x00FF00FF
			dst[i] = mix2(dst[i], qrb*255+0x00800080, qga*255+0x00800080, 255-qga>>16)
		}
	}
}
