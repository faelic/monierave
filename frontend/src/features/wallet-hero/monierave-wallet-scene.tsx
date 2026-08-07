"use client";

import { Canvas } from "@react-three/fiber";
import { Suspense, useEffect, useRef, useState } from "react";
import type { Group } from "three";

import { AnimationController } from "./components/animation-controller";
import { BankingCard } from "./components/banking-cards";
import { DigitalCardHolder } from "./components/digital-card-holder";
import { PaymentToken } from "./components/payment-token";
import { ResponsiveCamera } from "./components/responsive-camera";
import { StudioLighting } from "./components/studio-lighting";
import { walletResponsive } from "./config/wallet-config";

function SceneReady({ onReady }: { onReady: () => void }) {
  useEffect(() => {
    onReady();
  }, [onReady]);

  return null;
}

function WalletComposition() {
  const sceneRef = useRef<Group>(null);

  return (
    <>
      <ResponsiveCamera />
      <StudioLighting />
      <group
        name="sceneRoot"
        position={[0, -0.72, 0]}
        ref={sceneRef}
        scale={walletResponsive.groupScale.standard}
      >
        <group name="walletMotionRoot">
          <DigitalCardHolder />
          <BankingCard kind="supporting" />
          <BankingCard kind="secondary" />
          <BankingCard kind="primary" />

          <PaymentToken name="paymentTokenA" variant="off-white" />
          <PaymentToken name="paymentTokenB" variant="mint" />
        </group>

        <AnimationController sceneRef={sceneRef} />
      </group>
    </>
  );
}

export default function MonieraveWalletScene({
  onReady,
  onSceneFailure,
}: {
  onReady: () => void;
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
      aria-label="A Monierave digital wallet opens to reveal three payment cards while two transaction tokens move around it"
      className="relative size-full"
      role="img"
    >
      <Canvas
        camera={{
          fov: walletResponsive.camera.fov,
          position: [...walletResponsive.camera.position],
        }}
        dpr={walletResponsive.dpr}
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
        shadows="basic"
      >
        <Suspense fallback={null}>
          <WalletComposition />
          <SceneReady onReady={onReady} />
        </Suspense>
      </Canvas>
    </div>
  );
}
