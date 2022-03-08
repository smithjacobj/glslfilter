#version 330 core
#extension GL_ARB_separate_shader_objects : enable
#extension GL_ARB_explicit_uniform_location : enable
#extension GL_ARB_shading_language_420pack : enable

layout(location = 0) uniform ivec2 viewportSize;
layout(location = 1) in vec2 fragTexCoord;
layout(binding = 1) uniform sampler2D baseTexture;
layout(binding = 2) uniform sampler2D triggerTexture;

layout(location = 0) out vec4 fragColor;

vec2 texcoordMod(vec2 v, vec2 modulo) {
    return vec2(v.x - mod(v.x, modulo.x), v.y - mod(v.y, modulo.y));
}

vec4 runBack() {
    vec2 triggerSize = textureSize(triggerTexture, 0);
    vec2 triggerTexelSize = vec2(1.0 / triggerSize.x, 1.0 / triggerSize.y);
    vec2 lastGoodStart = texcoordMod(fragTexCoord, triggerTexelSize);
    lastGoodStart.x += triggerTexelSize.x / 2.0;
    lastGoodStart.y += triggerTexelSize.y / 2.0;
    vec2 lastGood = lastGoodStart;
    while (lastGood.y > 0 && texture(triggerTexture, lastGood).r < 1.0) {
        vec2 newLastGood = vec2(lastGood.x - triggerTexelSize.x, lastGood.y);
        if (newLastGood.x < 0) {
            newLastGood.x = lastGoodStart.x;
            newLastGood.y -= triggerTexelSize.y;
            if (texture(triggerTexture, newLastGood).r >= 1.0) {
                break;
            }
        }
        lastGood = newLastGood;
    }
    if (lastGood.y <= 0) {
        return vec4(vec3(0), 1.0);
    } else {
        lastGood = texcoordMod(lastGood, triggerTexelSize);
        return texture(baseTexture, lastGood);
    }
}

void main() {
    vec4 triggerColor = texture(triggerTexture, fragTexCoord);
    if (triggerColor.r < 1.0) {
        fragColor = runBack();
    } else {
        fragColor = texture(baseTexture, fragTexCoord);
    }
}