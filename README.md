# GLSL Image Filter
I've had the mind to create a filter that would make tiled blend passes simple since I published [Digital Evolution](https://www.deviantart.com/lapinlunaire/art/Digital-Evolution-106171666) 14 years ago. I recently came back around to it; I've a lot more experience with graphics and shaders now and it seemed sensible to make something using shader code.

I actually originally was going to write it in Vulkan, but I wanted to skip the boilerplate and the fact that you need to compile shader code to SPIR-V first - while SPIR-V is sensible, it complicates the code here. OpenGL lets me defer that functionality to the driver.

If you just want to try it out, build `glslfilter-glfw` and see `/demo/crt-singlestage`
