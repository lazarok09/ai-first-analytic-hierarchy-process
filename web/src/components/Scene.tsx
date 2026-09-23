"use client";

import { Canvas } from "@react-three/fiber";
import { Suspense } from "react";
import * as THREE from "three";
import { DeckEngine } from "./DeckEngine";

export function Scene() {
  return (
    <div className="pointer-events-none fixed inset-0 z-0" aria-hidden>
      <Canvas
        dpr={[1, 1.5]}
        gl={{
          antialias: true,
          alpha: true,
          powerPreference: "high-performance",
          toneMapping: THREE.ACESFilmicToneMapping,
          toneMappingExposure: 1.05,
        }}
        camera={{ position: [0, 1.1, 7.4], fov: 36, near: 0.1, far: 60 }}
      >
        <color attach="background" args={["#0a0c0f"]} />
        <fog attach="fog" args={["#0a0c0f", 11, 26]} />
        <Suspense fallback={null}>
          <DeckEngine />
        </Suspense>
      </Canvas>
    </div>
  );
}
