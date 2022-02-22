package main

var Radius float
var Color vec4

func Fragment(pos vec4, texCoord vec2, color vec4) vec4 {
    dist := distance(pos.xy, vec2(Radius + 1))

    if dist < Radius - 1 {
        return Color
    } else {
        // return vec4(1, 1, 0, 1)
        return vec4(0)
    }
}