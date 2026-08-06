"use client";

import { RoundedBox } from "@react-three/drei";
import { Canvas, useFrame } from "@react-three/fiber";
import { useEffect, useRef, useState } from "react";
import type { Group } from "three";

const vaultNodes: [number, number, number][] = [
  [-2.2, 1.1, 0.4],
  [2.35, 0.62, -0.2],
  [1.4, -1.85, 0.6],
];

function VaultAssembly() {
  const assembly = useRef<Group>(null);
  const card = useRef<Group>(null);

  useFrame(({ clock, pointer }, delta) => {
    if (!assembly.current || !card.current) {
      return;
    }

    const elapsed = clock.getElapsedTime();
    assembly.current.rotation.y +=
      (pointer.x * 0.12 - assembly.current.rotation.y) *
      Math.min(delta * 2.5, 1);
    assembly.current.rotation.x +=
      (-pointer.y * 0.08 - assembly.current.rotation.x) *
      Math.min(delta * 2.5, 1);
    card.current.position.y = Math.sin(elapsed * 0.75) * 0.09;
    card.current.rotation.z = -0.075 + Math.sin(elapsed * 0.5) * 0.018;
  });

  return (
    <group ref={assembly}>
      <mesh rotation={[Math.PI / 2.5, 0.18, 0.15]}>
        <torusGeometry args={[2.52, 0.035, 10, 96]} />
        <meshStandardMaterial
          color="#65d5ae"
          emissive="#24a779"
          emissiveIntensity={0.8}
          transparent
          opacity={0.58}
        />
      </mesh>
      <mesh rotation={[Math.PI / 2.1, -0.4, 0.62]}>
        <torusGeometry args={[2.14, 0.018, 8, 80, Math.PI * 1.55]} />
        <meshStandardMaterial
          color="#d8f1e7"
          emissive="#24a779"
          emissiveIntensity={0.45}
          transparent
          opacity={0.5}
        />
      </mesh>
      <mesh>
        <sphereGeometry args={[2.05, 32, 20]} />
        <meshPhysicalMaterial
          color="#d8f1e7"
          metalness={0.05}
          opacity={0.1}
          roughness={0.12}
          side={2}
          transparent
        />
      </mesh>

      <group ref={card} rotation={[0.08, -0.18, -0.075]}>
        <RoundedBox args={[3.28, 2.02, 0.16]} radius={0.16} smoothness={4}>
          <meshStandardMaterial
            color="#0e332c"
            metalness={0.38}
            roughness={0.3}
          />
        </RoundedBox>
        <mesh position={[-1.05, 0.48, 0.095]}>
          <boxGeometry args={[0.48, 0.34, 0.025]} />
          <meshStandardMaterial
            color="#b8c6bf"
            metalness={0.72}
            roughness={0.26}
          />
        </mesh>
        <mesh position={[0, -0.35, 0.1]}>
          <boxGeometry args={[2.5, 0.055, 0.025]} />
          <meshStandardMaterial
            color="#24a779"
            emissive="#24a779"
            emissiveIntensity={0.6}
          />
        </mesh>
        <mesh position={[-0.72, -0.65, 0.1]}>
          <boxGeometry args={[1.06, 0.045, 0.02]} />
          <meshStandardMaterial color="#d8f1e7" />
        </mesh>
      </group>

      {vaultNodes.map(([x, y, z]) => (
        <mesh key={`${x}-${y}`} position={[x, y, z]}>
          <sphereGeometry args={[0.115, 16, 12]} />
          <meshStandardMaterial
            color="#65d5ae"
            emissive="#24a779"
            emissiveIntensity={1.1}
          />
        </mesh>
      ))}
    </group>
  );
}

export default function VaultScene({
  onSceneFailure,
}: {
  onSceneFailure: () => void;
}) {
  const [visible, setVisible] = useState(
    typeof document === "undefined" || document.visibilityState === "visible",
  );

  useEffect(() => {
    const handleVisibility = () => {
      setVisible(document.visibilityState === "visible");
    };

    document.addEventListener("visibilitychange", handleVisibility);
    return () =>
      document.removeEventListener("visibilitychange", handleVisibility);
  }, []);

  return (
    <div
      aria-label="A virtual Monierave card protected by a transparent circular vault"
      className="size-full"
      role="img"
    >
      <Canvas
        camera={{ fov: 36, position: [0, 0.15, 7.4] }}
        dpr={[1, 1.5]}
        frameloop={visible ? "always" : "never"}
        gl={{
          alpha: true,
          antialias: true,
          powerPreference: "high-performance",
        }}
        onCreated={({ gl }) => {
          gl.domElement.addEventListener("webglcontextlost", onSceneFailure, {
            once: true,
          });
        }}
      >
        <ambientLight intensity={1.2} />
        <directionalLight
          color="#ffffff"
          intensity={3.2}
          position={[3, 4, 5]}
        />
        <pointLight color="#24a779" intensity={12} position={[-3, -1, 3]} />
        <pointLight color="#d8f1e7" intensity={7} position={[3, 1, 1]} />
        <VaultAssembly />
      </Canvas>
    </div>
  );
}
