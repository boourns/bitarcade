package scene

import (
	"github.com/vova616/chipmunk"
	"github.com/vova616/chipmunk/vect"
	"math/rand"
	"math"
	"encoding/json"
)

var ballRadius = 25
var ballMass = 1

type Scene struct {
	space       *chipmunk.Space
	balls       []*chipmunk.Shape
	staticLines []*chipmunk.Shape
}

type Box struct {
	X, Y, A vect.Float
}

type Line struct {
	Point1, Point2 vect.Vect
}

func (s *Scene) AddBall(x int, y int) {
	ball := chipmunk.NewCircle(vect.Vector_Zero, float32(ballRadius))
	ball.SetElasticity(0.95)

	body := chipmunk.NewBody(vect.Float(ballMass), ball.Moment(float32(ballMass)))
	body.SetPosition(vect.Vect{vect.Float(x), vect.Float(y)})
	body.SetAngle(vect.Float(rand.Float32() * 2 * math.Pi))
	body.AddShape(ball)

	s.space.AddBody(body)
	s.balls = append(s.balls, ball)
}

// step advances the physics engine and cleans up any balls that are off-screen
func (s *Scene) Step(dt float32) {
	s.space.Step(vect.Float(dt))

	for i := 0; i < len(s.balls); i++ {
		p := s.balls[i].Body.Position()
		if p.Y < -100 {
			s.space.RemoveBody(s.balls[i].Body)
			s.balls[i] = nil
			s.balls = append(s.balls[:i], s.balls[i+1:]...)
			i-- // consider same index again
		}
	}
}

func New() (*Scene) {
	s := &Scene{}
	s.space = chipmunk.NewSpace()
	s.space.Gravity = vect.Vect{0, -900}

	staticBody := chipmunk.NewBodyStatic()
	s.staticLines = []*chipmunk.Shape{
		chipmunk.NewSegment(vect.Vect{111.0, 280.0}, vect.Vect{407.0, 246.0}, 0),
		chipmunk.NewSegment(vect.Vect{407.0, 246.0}, vect.Vect{407.0, 343.0}, 0),
	}

	for _, segment := range s.staticLines {
		segment.SetElasticity(0.6)
		staticBody.AddShape(segment)
	}
	s.space.AddBody(staticBody)
	return s
}

func (s *Scene) Render() ([]byte, error) {
	message := make(map[string]interface{})
	boxes := make([]Box, len(s.balls))

	for i, x := range s.balls {
		boxes[i] = Box{x.Body.Position().X, x.Body.Position().Y, x.Body.Angle()}
	}

	lines := make([]Line, len(s.staticLines))
	for i, x := range s.staticLines {
		lines[i].Point1 = x.GetAsSegment().A
		lines[i].Point2 = x.GetAsSegment().B
	}

	message["boxes"] = boxes
	message["lines"] = lines

 	return json.Marshal(message)
}
