"use client";

import { Canvas } from "@react-three/fiber";
import {
  ACESFilmicToneMapping,
  CatmullRomCurve3,
  Color,
  Quaternion,
  SRGBColorSpace,
  Vector3,
} from "three";

import {
  carrierSegments,
  pipelineBrand,
  pipelineCamera,
  pipelinePaths,
  supportSegments,
  type PipelinePoint,
} from "./pipeline-config";

const unitY = new Vector3(0, 1, 0);
const unitZ = new Vector3(0, 0, 1);

function vector([x, y, z]: PipelinePoint) {
  return new Vector3(x, y, z);
}

function makeCurve(points: readonly PipelinePoint[]) {
  return new CatmullRomCurve3(
    points.map(vector),
    false,
    "catmullrom",
    0.38,
  );
}

const curves = {
  lowerRail: makeCurve(pipelinePaths.lowerRail),
  middleRail: makeCurve(pipelinePaths.middleRail),
  rearDrop: makeCurve(pipelinePaths.rearDrop),
  rearLoop: makeCurve(pipelinePaths.rearLoop),
  upperRail: makeCurve(pipelinePaths.upperRail),
};

function PipelineTube({
  color,
  curve,
  metalness,
  opacity = 1,
  radius,
  roughness,
}: {
  color: string;
  curve: CatmullRomCurve3;
  metalness: number;
  opacity?: number;
  radius: number;
  roughness: number;
}) {
  return (
    <mesh castShadow opacity={opacity} receiveShadow>
      <tubeGeometry args={[curve, 128, radius, 20, false]} />
      <meshStandardMaterial
        color={color}
        metalness={metalness}
        opacity={opacity}
        roughness={roughness}
        transparent={opacity < 1}
      />
    </mesh>
  );
}

function CylinderSegment({
  color,
  end,
  metalness,
  opacity = 1,
  radius,
  roughness,
  start,
}: {
  color: string;
  end: PipelinePoint;
  metalness: number;
  opacity?: number;
  radius: number;
  roughness: number;
  start: PipelinePoint;
}) {
  const startVector = vector(start);
  const endVector = vector(end);
  const direction = endVector.clone().sub(startVector);
  const midpoint = startVector.clone().add(endVector).multiplyScalar(0.5);
  const quaternion = new Quaternion().setFromUnitVectors(
    unitY,
    direction.clone().normalize(),
  );

  return (
    <mesh
      castShadow
      position={midpoint}
      quaternion={quaternion}
      receiveShadow
    >
      <cylinderGeometry args={[radius, radius, direction.length(), 24, 1]} />
      <meshStandardMaterial
        color={color}
        metalness={metalness}
        opacity={opacity}
        roughness={roughness}
        transparent={opacity < 1}
      />
    </mesh>
  );
}

function Coupling({
  point,
  tangent = [0.55, 0.83, 0],
}: {
  point: PipelinePoint;
  tangent?: PipelinePoint;
}) {
  const quaternion = new Quaternion().setFromUnitVectors(
    unitZ,
    vector(tangent).normalize(),
  );

  return (
    <mesh castShadow position={vector(point)} quaternion={quaternion}>
      <torusGeometry args={[0.235, 0.045, 14, 48]} />
      <meshStandardMaterial
        color={pipelineBrand.warmMetal}
        metalness={0.84}
        roughness={0.2}
      />
    </mesh>
  );
}

function NodeLight({
  point,
  scale = 1,
}: {
  point: PipelinePoint;
  scale?: number;
}) {
  return (
    <group position={vector(point)} scale={scale}>
      <pointLight
        color={pipelineBrand.nodeBlue}
        distance={1.1}
        intensity={1.35}
      />
      <mesh castShadow>
        <boxGeometry args={[0.11, 0.11, 0.11]} />
        <meshStandardMaterial
          color="#DDE7FF"
          emissive={pipelineBrand.nodeBlue}
          emissiveIntensity={1.7}
          metalness={0.22}
          roughness={0.18}
        />
      </mesh>
    </group>
  );
}

function Carrier() {
  return (
    <group position={[-0.45, 0, 0]}>
      {carrierSegments.map((segment, index) => (
        <CylinderSegment key={index} {...segment} />
      ))}
      <Coupling point={[-1.95, -1.18, 1.15]} />
      <Coupling point={[-0.62, -0.3, 1.15]} />
      <Coupling point={[0.82, 0.68, 1.15]} />

      {[-0.28, -0.17, -0.06, 0.05].map((offset) => (
        <mesh
          key={offset}
          position={[offset - 0.18, offset * 0.67 + 0.03, 1.36]}
        >
          <sphereGeometry args={[0.026, 10, 10]} />
          <meshBasicMaterial color="#E6E5E1" />
        </mesh>
      ))}

      <group position={[0.82, 0.68, 1.52]}>
        <pointLight color="#FFF4D8" distance={2.4} intensity={6.5} />
        <mesh castShadow>
          <sphereGeometry args={[0.235, 24, 24]} />
          <meshStandardMaterial
            color="#FFF7E7"
            emissive="#FFE4B5"
            emissiveIntensity={1.15}
            metalness={0.16}
            roughness={0.2}
          />
        </mesh>
      </group>
    </group>
  );
}

function Reflections() {
  return (
    <group position={[-0.45, -3.56, -1.8]} scale={[1, -0.48, 1]}>
      {carrierSegments.slice(0, 4).map((segment, index) => (
        <CylinderSegment
          {...segment}
          color={index % 2 === 0 ? pipelineBrand.silverDark : "#08090B"}
          key={index}
          metalness={0.45}
          opacity={0.15}
          roughness={0.5}
        />
      ))}
      <PipelineTube
        color="#0A0B0E"
        curve={curves.middleRail}
        metalness={0.18}
        opacity={0.12}
        radius={0.13}
        roughness={0.68}
      />
    </group>
  );
}

function PipelineComposition() {
  return (
    <group position={[0.42, 0.5, 0]} scale={[0.86, 0.84, 1]}>
      <PipelineTube
        color={pipelineBrand.silver}
        curve={curves.rearLoop}
        metalness={0.9}
        radius={0.265}
        roughness={0.16}
      />
      <PipelineTube
        color={pipelineBrand.silverDark}
        curve={curves.rearDrop}
        metalness={0.65}
        radius={0.245}
        roughness={0.28}
      />
      <Coupling point={[-3.53, 1.45, -1.35]} tangent={[-0.15, 0.98, 0]} />
      <Coupling point={[-0.14, 1.95, -1.35]} tangent={[0.98, 0.14, 0]} />

      {supportSegments.map((segment, index) => (
        <CylinderSegment
          color={pipelineBrand.black}
          key={index}
          metalness={0.52}
          radius={0.105}
          roughness={0.3}
          {...segment}
        />
      ))}

      <PipelineTube
        color={pipelineBrand.black}
        curve={curves.lowerRail}
        metalness={0.58}
        radius={0.145}
        roughness={0.24}
      />
      <PipelineTube
        color={pipelineBrand.graphiteEdge}
        curve={curves.upperRail}
        metalness={0.7}
        radius={0.205}
        roughness={0.2}
      />
      <PipelineTube
        color={pipelineBrand.graphite}
        curve={curves.middleRail}
        metalness={0.66}
        radius={0.19}
        roughness={0.18}
      />

      <NodeLight point={[-2.95, -0.02, 0]} />
      <NodeLight point={[-1.45, 0.45, 0]} />
      <NodeLight point={[1.92, 0.68, 0]} />
      <NodeLight point={[3.05, -1.66, -0.35]} />

      <Carrier />
      <Reflections />
    </group>
  );
}

export default function PipelineScene({ onReady }: { onReady?: () => void }) {
  return (
    <Canvas
      camera={{
        far: 40,
        near: 0.1,
        position: [...pipelineCamera.position],
        zoom: pipelineCamera.zoom,
      }}
      dpr={pipelineCamera.dpr}
      frameloop="demand"
      gl={{
        alpha: true,
        antialias: true,
        powerPreference: "high-performance",
        preserveDrawingBuffer: true,
      }}
      onCreated={({ camera, gl }) => {
        camera.lookAt(0, 0, 0);
        gl.outputColorSpace = SRGBColorSpace;
        gl.toneMapping = ACESFilmicToneMapping;
        gl.toneMappingExposure = 1.12;
        gl.setClearColor(new Color(pipelineBrand.background), 0);
        onReady?.();
      }}
      orthographic
      shadows
    >
      <ambientLight intensity={0.24} />
      <directionalLight
        castShadow
        color="#F7F5EF"
        intensity={4.8}
        position={[-4.2, 6.5, 7]}
        shadow-mapSize-height={512}
        shadow-mapSize-width={512}
      />
      <spotLight
        angle={0.42}
        color="#CBD8F8"
        intensity={3.2}
        penumbra={0.9}
        position={[4.8, 4.2, 6]}
      />
      <pointLight
        color="#B07859"
        distance={9}
        intensity={6}
        position={[0.6, -0.1, 4]}
      />
      <PipelineComposition />
    </Canvas>
  );
}
