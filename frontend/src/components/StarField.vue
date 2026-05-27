<template>
  <canvas ref="canvas" class="starfield" aria-hidden="true" />
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'

const canvas = ref<HTMLCanvasElement | null>(null)

interface Star {
  x: number
  y: number
  size: number
  opacity: number
  opacityDelta: number
  vx: number
  vy: number
  // occasional "chaos kick" timer
  kickIn: number
}

let raf = 0

function rand(min: number, max: number) {
  return min + Math.random() * (max - min)
}

function makeStars(w: number, h: number, count: number): Star[] {
  return Array.from({ length: count }, () => {
    const size = Math.random() < 0.15 ? rand(1.5, 2.8) : rand(0.4, 1.2)
    // base speed scales inversely with size (small stars drift faster)
    const speed = rand(0.04, 0.22) / size
    const angle = rand(0, Math.PI * 2)
    return {
      x: rand(0, w),
      y: rand(0, h),
      size,
      opacity: rand(0.3, 1.0),
      opacityDelta: rand(0.002, 0.006) * (Math.random() < 0.5 ? 1 : -1),
      vx: Math.cos(angle) * speed,
      vy: Math.sin(angle) * speed,
      kickIn: Math.floor(rand(60, 400)),
    }
  })
}

onMounted(() => {
  const el = canvas.value!
  const ctx = el.getContext('2d')!

  let w = 0
  let h = 0
  let stars: Star[] = []

  function resize() {
    w = window.innerWidth
    h = window.innerHeight
    el.width = w
    el.height = h
    // re-scatter stars on resize so they fill the new size
    stars = makeStars(w, h, 200)
  }

  resize()
  window.addEventListener('resize', resize)

  function tick() {
    ctx.clearRect(0, 0, w, h)

    for (const s of stars) {
      // move
      s.x += s.vx
      s.y += s.vy

      // wrap around edges
      if (s.x < -2) s.x = w + 2
      else if (s.x > w + 2) s.x = -2
      if (s.y < -2) s.y = h + 2
      else if (s.y > h + 2) s.y = -2

      // twinkle
      s.opacity += s.opacityDelta
      if (s.opacity > 1) { s.opacity = 1; s.opacityDelta *= -1 }
      else if (s.opacity < 0.15) { s.opacity = 0.15; s.opacityDelta *= -1 }

      // periodic chaos kick — randomly change direction
      s.kickIn--
      if (s.kickIn <= 0) {
        const speed = rand(0.04, 0.28) / s.size
        const angle = rand(0, Math.PI * 2)
        s.vx = Math.cos(angle) * speed
        s.vy = Math.sin(angle) * speed
        s.kickIn = Math.floor(rand(80, 500))
      }

      // draw
      const hue = Math.random() < 0.12 ? '220, 200, 255' : '255, 255, 255'
      ctx.beginPath()
      ctx.arc(s.x, s.y, s.size, 0, Math.PI * 2)
      ctx.fillStyle = `rgba(${hue}, ${s.opacity})`
      ctx.fill()
    }

    raf = requestAnimationFrame(tick)
  }

  raf = requestAnimationFrame(tick)

  onUnmounted(() => {
    cancelAnimationFrame(raf)
    window.removeEventListener('resize', resize)
  })
})
</script>

<style scoped>
.starfield {
  position: fixed;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 0;
}
</style>
