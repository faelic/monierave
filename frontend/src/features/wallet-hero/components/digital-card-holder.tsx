import { RoundedBox } from "@react-three/drei";

import {
  monieraveBrand,
  walletMaterials,
  walletTransforms,
} from "../config/wallet-config";

function ShellPanel({ height, liningZ }: { height: number; liningZ: number }) {
  const holder = walletTransforms.holder;

  return (
    <group position={[0, height / 2, 0]}>
      <RoundedBox
        args={[holder.panelWidth, height, holder.panelDepth]}
        castShadow
        radius={0.2}
        receiveShadow
        smoothness={6}
      >
        <meshPhysicalMaterial
          clearcoat={0.08}
          color={monieraveBrand.monieraveOffWhite}
          metalness={walletMaterials.holderExterior.metalness}
          roughness={walletMaterials.holderExterior.roughness}
        />
      </RoundedBox>
      <RoundedBox
        args={[holder.panelWidth - 0.18, height - 0.17, 0.035]}
        position={[0, 0, liningZ]}
        radius={0.16}
        smoothness={5}
      >
        <meshStandardMaterial
          color={monieraveBrand.monieraveForest}
          metalness={walletMaterials.holderInterior.metalness}
          roughness={walletMaterials.holderInterior.roughness}
        />
      </RoundedBox>
    </group>
  );
}

export function DigitalCardHolder() {
  const holder = walletTransforms.holder;
  const frontHeight = 1.62;

  return (
    <group name="digitalCardHolder">
      <RoundedBox
        args={[holder.panelWidth + 0.12, holder.baseHeight, holder.baseDepth]}
        castShadow
        position={[0, holder.hingeY - 0.08, 0]}
        radius={0.14}
        receiveShadow
        smoothness={5}
      >
        <meshPhysicalMaterial
          clearcoat={0.08}
          color={monieraveBrand.monieraveOffWhite}
          metalness={walletMaterials.holderExterior.metalness}
          roughness={walletMaterials.holderExterior.roughness}
        />
      </RoundedBox>

      <group
        name="backPanelPivot"
        position={[0, holder.hingeY, -holder.baseDepth / 2 + 0.12]}
      >
        <ShellPanel height={holder.panelHeight} liningZ={0.1} />
      </group>

      <group
        name="frontPanelPivot"
        position={[0, holder.hingeY, holder.baseDepth / 2 - 0.12]}
      >
        <ShellPanel height={frontHeight} liningZ={-0.1} />
        <RoundedBox
          args={[1.08, 0.075, 0.045]}
          position={[0.9, 0.26, -0.12]}
          radius={0.03}
          smoothness={4}
        >
          <meshStandardMaterial
            color={monieraveBrand.monieraveMint}
            metalness={0.12}
            roughness={0.42}
          />
        </RoundedBox>
      </group>

      <RoundedBox
        args={[holder.panelWidth - 0.28, 0.1, 0.12]}
        position={[0, holder.hingeY + 0.13, 0.57]}
        radius={0.04}
        smoothness={4}
      >
        <meshStandardMaterial
          color={monieraveBrand.monieraveSage}
          metalness={0.1}
          roughness={0.48}
        />
      </RoundedBox>

      <mesh
        castShadow
        position={[-1.72, holder.hingeY - 0.06, 0.59]}
        rotation={[Math.PI / 2, 0, 0]}
      >
        <cylinderGeometry args={[0.11, 0.11, 0.055, 28]} />
        <meshStandardMaterial
          color={monieraveBrand.monieraveDark}
          metalness={walletMaterials.satin.metalness}
          roughness={walletMaterials.satin.roughness}
        />
      </mesh>
    </group>
  );
}
