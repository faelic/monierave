import { RoundedBox, useTexture } from "@react-three/drei";
import { useEffect, useMemo } from "react";
import { LinearFilter, LinearMipmapLinearFilter, SRGBColorSpace } from "three";

import {
  monieraveBrand,
  walletAssets,
  walletMaterials,
} from "../config/wallet-config";

type CardKind = keyof typeof walletAssets.textures;

const cardColors: Record<CardKind, string> = {
  primary: monieraveBrand.monieraveForest,
  secondary: monieraveBrand.monieraveOffWhite,
  supporting: monieraveBrand.monieraveDark,
};

const cardNames: Record<CardKind, string> = {
  primary: "primaryCardPivot",
  secondary: "secondaryCardPivot",
  supporting: "supportingCardPivot",
};

export function BankingCard({ kind }: { kind: CardKind }) {
  const sourceTexture = useTexture(walletAssets.textures[kind]);
  const texture = useMemo(() => {
    const configuredTexture = sourceTexture.clone();
    configuredTexture.colorSpace = SRGBColorSpace;
    configuredTexture.magFilter = LinearFilter;
    configuredTexture.minFilter = LinearMipmapLinearFilter;
    configuredTexture.anisotropy = 4;
    configuredTexture.needsUpdate = true;
    return configuredTexture;
  }, [sourceTexture]);

  useEffect(() => () => texture.dispose(), [texture]);

  return (
    <group name={cardNames[kind]}>
      <RoundedBox
        args={[3.38, 2.08, 0.12]}
        castShadow
        radius={0.17}
        receiveShadow
        smoothness={5}
      >
        <meshPhysicalMaterial
          clearcoat={0.06}
          color={cardColors[kind]}
          metalness={walletMaterials.card.metalness}
          roughness={walletMaterials.card.roughness}
        />
      </RoundedBox>
      <mesh position={[0, 0, 0.064]}>
        <planeGeometry args={[3.3, 2]} />
        <meshStandardMaterial
          map={texture}
          metalness={walletMaterials.card.metalness}
          roughness={walletMaterials.card.roughness}
          transparent
        />
      </mesh>
    </group>
  );
}
