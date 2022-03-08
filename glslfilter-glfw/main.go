package main

import (
	"image"
	_ "image/png"
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/smithjacobj/glslfilter"
	"github.com/smithjacobj/glslfilter/util"
)

const AppName = "GLSL Filter"

func init() {
	// GLFW event handling must run on the main OS thread
	runtime.LockOSThread()
}

func main() {
	util.Invariant(glfw.Init())
	defer glfw.Terminate()

	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	definition, err := glslfilter.LoadDefinitionFromStdin()
	util.Invariant(err)

	window, err := glfw.CreateWindow(definition.Render.Width, definition.Render.Height, AppName, nil, nil)
	util.Invariant(err)
	window.MakeContextCurrent()

	engine, err := glslfilter.NewEngine(image.Rect(0, 0, definition.Render.Width, definition.Render.Height), true)
	util.Invariant(err)

	stages := []*glslfilter.FilterStage{}
	for _, stageDefinition := range definition.Stages {
		fragmentShaderSource, err := glslfilter.LoadFragmentShader(stageDefinition.FragmentShaderPath)
		util.Invariant(err)

		textures := []glslfilter.Texture{}
		for _, textureDefinition := range stageDefinition.Textures {
			textureRGBA, err := glslfilter.LoadTextureData(textureDefinition.Path)
			util.Invariant(err)
			textures = append(textures, glslfilter.Texture{Data: textureRGBA, Filter: int32(textureDefinition.Filter)})
		}

		stage, err := glslfilter.NewFilterStage(fragmentShaderSource, textures)
		util.Invariant(err)

		stages = append(stages, stage)
	}

	err = engine.Init(stages)
	util.Invariant(err)

	engine.Render()

	window.SwapBuffers()
	for !window.ShouldClose() {
		glfw.PollEvents()
	}
}
