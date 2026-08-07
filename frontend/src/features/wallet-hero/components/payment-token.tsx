import { monieraveBrand, walletMaterials } from "../config/wallet-config";

export function PaymentToken({
  name,
  variant,
}: {
  name: "paymentTokenA" | "paymentTokenB";
  variant: "mint" | "off-white";
}) {
  const isMint = variant === "mint";
  const faceColor = isMint
    ? monieraveBrand.monieraveMint
    : monieraveBrand.monieraveOffWhite;
  const markColor = isMint
    ? monieraveBrand.monieraveOffWhite
    : monieraveBrand.monieraveMint;

  return (
    <group name={name} scale={0}>
      <mesh castShadow rotation={[Math.PI / 2, 0, 0]}>
        <cylinderGeometry args={[0.28, 0.28, 0.105, 36]} />
        <meshPhysicalMaterial
          clearcoat={0.16}
          color={faceColor}
          metalness={walletMaterials.token.metalness}
          roughness={walletMaterials.token.roughness}
        />
      </mesh>
      <mesh position={[0, 0, 0.059]}>
        <ringGeometry args={[0.135, 0.17, 32]} />
        <meshBasicMaterial color={markColor} />
      </mesh>
      <mesh position={[0.072, 0.008, 0.063]} rotation={[0, 0, -0.7]}>
        <boxGeometry args={[0.145, 0.037, 0.014]} />
        <meshBasicMaterial color={markColor} />
      </mesh>
    </group>
  );
}
