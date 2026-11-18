package gestures

import "hexago/ui/geom"

// Подробности жестов. Position — в координатах экрана, Local — в координатах виджета,
// на котором распознан жест; всё в dp.

// TapDownDetails — палец опустился, и это, похоже, нажатие.
type TapDownDetails struct {
	Position, Local geom.Offset
}

// TapUpDetails — палец отпущен, нажатие состоялось.
type TapUpDetails struct {
	Position, Local geom.Offset
}

// LongPressStartDetails — палец продержали достаточно долго.
type LongPressStartDetails struct {
	Position, Local geom.Offset
}

// LongPressMoveUpdateDetails — после начала долгого нажатия палец двигается.
type LongPressMoveUpdateDetails struct {
	Position, Local  geom.Offset
	OffsetFromOrigin geom.Offset // сдвиг от точки, где началось долгое нажатие
}

// LongPressEndDetails — палец отпущен после долгого нажатия.
type LongPressEndDetails struct {
	Position, Local geom.Offset
	Velocity        geom.Offset // dp/с
}

// DragStartDetails — перетаскивание началось (в точке, где палец опустился).
type DragStartDetails struct {
	Position, Local geom.Offset
}

// DragUpdateDetails — палец сдвинулся во время перетаскивания.
type DragUpdateDetails struct {
	Position, Local geom.Offset
	Delta           geom.Offset // сдвиг с прошлого события; у вертикального — только Y, у горизонтального — только X
	PrimaryDelta    float64     // сдвиг вдоль оси вертикального или горизонтального перетаскивания; у свободного — 0
}

// DragEndDetails — палец отпущен.
type DragEndDetails struct {
	Velocity        geom.Offset // скорость броска, dp/с; ноль, если палец остановился
	PrimaryVelocity float64     // скорость вдоль оси вертикального или горизонтального перетаскивания
}

// ScaleStartDetails — начался масштаб (или перемещение двумя и более пальцами).
type ScaleStartDetails struct {
	FocalPoint, LocalFocalPoint geom.Offset // середина между пальцами
	PointerCount                int
}

// ScaleUpdateDetails — пальцы сдвинулись.
type ScaleUpdateDetails struct {
	FocalPoint, LocalFocalPoint geom.Offset
	FocalPointDelta             geom.Offset // сдвиг середины с прошлого события
	Scale                       float64     // во сколько раз изменилось расстояние между пальцами от начала
	HorizontalScale             float64
	VerticalScale               float64
	Rotation                    float64 // поворот первых двух пальцев от начала, в радианах
	PointerCount                int
}

// ScaleEndDetails — все пальцы отпущены.
type ScaleEndDetails struct {
	Velocity     geom.Offset // скорость середины, dp/с
	PointerCount int
}
