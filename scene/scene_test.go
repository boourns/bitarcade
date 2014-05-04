package scene

import (
  "testing"
  "fmt"
)

func TestSceneRenderable(t *testing.T) {
   scene := New()
   scene.AddBall(100, 600)
   for i:= 0; i < 60; i++ {
     scene.Step(1.0/60.0)
   }
   state, err := scene.Render()
   if err != nil {
	t.Fatalf("Error rendering scene: %v", err)
   }

   fmt.Printf("scene state %v\n", string(state))
}


