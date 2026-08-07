import { ContactShadows } from "@react-three/drei";

import { monieraveBrand, walletResponsive } from "../config/wallet-config";

export function StudioLighting() {
  return (
    <>
      <ambientLight intensity={1.25} />
      <directionalLight
        castShadow
        color={monieraveBrand.monieraveOffWhite}
        intensity={3.8}
        position={[-4.5, 6, 5]}
        shadow-mapSize-height={walletResponsive.shadowMapSize}
        shadow-mapSize-width={walletResponsive.shadowMapSize}
      />
      <directionalLight
        color={monieraveBrand.monieraveSage}
        intensity={1.5}
        position={[0, 1.5, 5]}
      />
      <pointLight
        color={monieraveBrand.monieraveMint}
        intensity={6}
        position={[4, 2.5, 2.5]}
      />
      <ContactShadows
        blur={2.8}
        far={4}
        opacity={0.2}
        position={[0, -2.05, -0.2]}
        resolution={walletResponsive.shadowMapSize}
        scale={8}
      />
    </>
  );
}
