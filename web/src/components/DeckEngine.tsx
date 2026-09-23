"use client";

/**
 * 3D proposal: Judgment Deck
 * A fan of floating “judgment cards” (pairwise diffs / proposals).
 * Scroll: idle fan → shuffle (fake) → neat stack (files) → ghost peel
 * (agent proposal) → CR heat on one card → report plate → commit slam.
 */
import { useFrame } from "@react-three/fiber";
import { useMemo, useRef } from "react";
import * as THREE from "three";
import { useScrollState } from "@/lib/scroll-store";
import type { SectionId } from "@/lib/site";

const COUNT = 9;

const STAGE: Record<SectionId, number> = {
  hook: 0,
  lie: 1,
  stance: 2,
  agent: 3,
  proof: 4,
  close: 5,
};

const COL = {
  card: "#1a222e",
  cardEdge: "#3dd6c6",
  warn: "#ff6b7a",
  slab: "#d5dbe4",
  core: "#3dd6c6",
};

function stageBlend(active: SectionId, progress: number) {
  return STAGE[active] + progress * 0.22;
}

function fanPose(i: number, t: number) {
  const mid = (COUNT - 1) / 2;
  const u = (i - mid) / mid;
  return {
    x: u * 2.4,
    y: 0.15 + Math.sin(t * 0.8 + i) * 0.06,
    z: -Math.abs(u) * 0.35,
    ry: -u * 0.55,
    rx: -0.35,
  };
}

function stackPose(i: number) {
  return {
    x: 0,
    y: i * 0.07,
    z: 0,
    ry: 0,
    rx: -0.55,
  };
}

function commitPose(i: number) {
  // Collapse into a dense block, then lift as rank steps
  const row = i % 3;
  const col = Math.floor(i / 3);
  return {
    x: (col - 1) * 0.85,
    y: row * 0.55,
    z: 0,
    ry: 0,
    rx: 0,
  };
}

export function DeckEngine() {
  const { progress, active } = useScrollState();
  const root = useRef<THREE.Group>(null);
  const cards = useRef<THREE.Group>(null);
  const slab = useRef<THREE.Mesh>(null);
  const core = useRef<THREE.Mesh>(null);
  const ghosts = useRef<(THREE.Mesh | null)[]>([]);
  const pulse = useRef(0);

  const camTarget = useMemo(() => new THREE.Vector3(), []);
  const look = useMemo(() => new THREE.Vector3(0, 0.4, 0), []);
  const colorA = useMemo(() => new THREE.Color(), []);
  const colorB = useMemo(() => new THREE.Color(), []);
  const tmpQ = useMemo(() => new THREE.Quaternion(), []);
  const euler = useMemo(() => new THREE.Euler(), []);

  useFrame((state, delta) => {
    pulse.current += delta;
    const t = pulse.current;
    const stage = stageBlend(active, progress);
    const cam = state.camera as THREE.PerspectiveCamera;

    const scatter = Math.max(0, Math.min(1, stage - 0.5));
    const settle = Math.max(0, Math.min(1, stage - 1.35));
    const agent = Math.max(0, Math.min(1, stage - 2.45));
    const heat = Math.max(0, Math.min(1, stage - 3.2));
    const proof = Math.max(0, Math.min(1, stage - 3.5));
    const close = Math.max(0, Math.min(1, stage - 4.4));

    camTarget.set(
      Math.sin(stage * 0.35) * 0.8,
      1.1 + close * 0.4,
      7.4 - stage * 0.3,
    );
    cam.position.lerp(camTarget, 1 - Math.exp(-delta * 2.2));
    look.set(0, 0.35 + close * 0.3, 0);
    cam.lookAt(look);

    if (root.current) {
      root.current.rotation.y = Math.sin(t * 0.12) * 0.08;
    }

    if (core.current) {
      core.current.rotation.y += delta * 0.4;
      core.current.rotation.x = Math.sin(t * 0.5) * 0.2;
      const cm = core.current.material as THREE.MeshStandardMaterial;
      cm.emissiveIntensity = 0.35 + heat * 0.4 + close * 0.2;
      core.current.scale.setScalar(0.85 + settle * 0.15 - scatter * 0.1);
    }

    if (cards.current) {
      cards.current.children.forEach((child, i) => {
        if (!(child instanceof THREE.Mesh)) return;

        const fan = fanPose(i, t);
        const stack = stackPose(i);
        const commit = commitPose(i);

        // Blend poses across beats
        let px = fan.x;
        let py = fan.y;
        let pz = fan.z;
        let rx = fan.rx;
        let ry = fan.ry;

        if (scatter > 0.05) {
          // shuffle chaos
          px += Math.sin(t * 5 + i * 2) * scatter * (1.2 - settle) * 1.4;
          py += Math.cos(t * 4 + i) * scatter * (1.2 - settle) * 0.9;
          pz += Math.sin(t * 3.5 + i) * scatter * (1.2 - settle) * 1.1;
          ry += Math.sin(t * 3 + i) * scatter * (1 - settle);
        }

        // settle → stack
        const toStack = settle;
        px = THREE.MathUtils.lerp(px, stack.x, toStack);
        py = THREE.MathUtils.lerp(py, stack.y + 0.4, toStack);
        pz = THREE.MathUtils.lerp(pz, stack.z, toStack);
        rx = THREE.MathUtils.lerp(rx, stack.rx, toStack);
        ry = THREE.MathUtils.lerp(ry, stack.ry, toStack);

        // agent: peel top cards outward
        if (agent > 0.1 && agent < 0.95 && i >= COUNT - 3) {
          const peel = (i - (COUNT - 3)) * 0.35 + 0.4;
          px += peel * agent * 1.8;
          py += agent * 0.6;
          ry -= agent * 0.5;
        }

        // close → commit grid
        px = THREE.MathUtils.lerp(px, commit.x, close);
        py = THREE.MathUtils.lerp(py, commit.y, close);
        pz = THREE.MathUtils.lerp(pz, commit.z, close);
        rx = THREE.MathUtils.lerp(rx, commit.rx, close);
        ry = THREE.MathUtils.lerp(ry, commit.ry, close);

        child.position.x = THREE.MathUtils.damp(child.position.x, px, 4, delta);
        child.position.y = THREE.MathUtils.damp(child.position.y, py, 4, delta);
        child.position.z = THREE.MathUtils.damp(child.position.z, pz, 4, delta);

        euler.set(rx, ry, 0);
        tmpQ.setFromEuler(euler);
        child.quaternion.slerp(tmpQ, 1 - Math.exp(-delta * 4));

        const mat = child.material as THREE.MeshStandardMaterial;
        let hex = COL.card;
        let em = "#0c1218";
        let emI = 0.06;

        if (agent > 0.15 && i >= COUNT - 3) {
          hex = COL.cardEdge;
          em = COL.cardEdge;
          emI = 0.3 + Math.sin(t * 5 + i) * 0.1;
        }
        if (heat > 0.4 && i === 2) {
          hex = COL.warn;
          em = COL.warn;
          emI = 0.45 + Math.sin(t * 7) * 0.2;
        }
        if (close > 0.3) {
          hex = COL.cardEdge;
          em = COL.cardEdge;
          emI = 0.2 + close * 0.25;
        }
        if (proof > 0.3 && i === 0) {
          emI = Math.max(emI, 0.25);
        }

        mat.color.lerp(colorA.set(hex), 1 - Math.exp(-delta * 5));
        mat.emissive.lerp(colorB.set(em), 1 - Math.exp(-delta * 5));
        mat.emissiveIntensity = THREE.MathUtils.damp(mat.emissiveIntensity, emI, 5, delta);
      });
    }

    if (slab.current) {
      const v = Math.max(0, Math.min(1, (stage - 4.25) / 1.0));
      slab.current.visible = v > 0.04;
      slab.current.position.set(2.8, 0.9 + v * 0.4, 0.6);
      slab.current.rotation.set(0.2, -0.5, 0.05);
      const m = slab.current.material as THREE.MeshStandardMaterial;
      m.opacity = v * (1 - close * 0.6) * 0.95;
    }

    ghosts.current.forEach((mesh, i) => {
      if (!mesh) return;
      const v = Math.max(0, Math.min(1, (stage - 2.4) / 1.1));
      mesh.visible = v > 0.04 && v < 0.9;
      const orbit = t * 0.7 + i * 2.1;
      mesh.position.set(
        Math.cos(orbit) * 3.2,
        1.4 + Math.sin(orbit * 1.3) * 0.4,
        Math.sin(orbit) * 2.2,
      );
      mesh.rotation.y += delta * 0.8;
      (mesh.material as THREE.MeshStandardMaterial).opacity = v * 0.45;
    });
  });

  return (
    <>
      <ambientLight intensity={0.4} />
      <directionalLight position={[4, 7, 3]} intensity={1.05} color="#e8eef5" />
      <pointLight position={[-3, 2, 2]} intensity={0.7} color="#3dd6c6" distance={12} />

      <group ref={root}>
        {/* Soft ground ring */}
        <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -0.85, 0]}>
          <ringGeometry args={[2.4, 2.5, 64]} />
          <meshBasicMaterial color={COL.cardEdge} transparent opacity={0.28} />
        </mesh>
        <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -0.86, 0]}>
          <ringGeometry args={[3.3, 3.38, 64]} />
          <meshBasicMaterial color="#3a4555" transparent opacity={0.3} />
        </mesh>

        {/* Core — judgment nucleus */}
        <mesh ref={core} position={[0, 0.35, 0]}>
          <octahedronGeometry args={[0.32, 0]} />
          <meshStandardMaterial
            color={COL.core}
            emissive={COL.core}
            emissiveIntensity={0.4}
            roughness={0.35}
            metalness={0.3}
            transparent
            opacity={0.85}
          />
        </mesh>

        <group ref={cards}>
          {Array.from({ length: COUNT }).map((_, i) => {
            const fan = fanPose(i, 0);
            return (
              <mesh
                key={i}
                position={[fan.x, fan.y, fan.z]}
                rotation={[fan.rx, fan.ry, 0]}
              >
                <boxGeometry args={[1.35, 1.85, 0.045]} />
                <meshStandardMaterial
                  color={COL.card}
                  emissive="#0c1218"
                  emissiveIntensity={0.06}
                  roughness={0.45}
                  metalness={0.2}
                />
              </mesh>
            );
          })}
        </group>

        <mesh ref={slab} visible={false}>
          <boxGeometry args={[1.4, 1.9, 0.06]} />
          <meshStandardMaterial
            color={COL.slab}
            roughness={0.4}
            metalness={0.12}
            transparent
            opacity={0}
          />
        </mesh>

        {[0, 1].map((i) => (
          <mesh
            key={`ghost-${i}`}
            ref={(el) => {
              ghosts.current[i] = el;
            }}
            visible={false}
          >
            <boxGeometry args={[0.9, 1.2, 0.03]} />
            <meshStandardMaterial
              color={COL.cardEdge}
              emissive={COL.cardEdge}
              emissiveIntensity={0.45}
              transparent
              opacity={0}
              depthWrite={false}
            />
          </mesh>
        ))}
      </group>
    </>
  );
}
