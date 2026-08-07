import { PerspectiveCamera } from "@react-three/drei";
import { useLayoutEffect, useRef } from "react";
import type { PerspectiveCamera as PerspectiveCameraType } from "three";

import { walletResponsive } from "../config/wallet-config";

export function ResponsiveCamera() {
  const cameraRef = useRef<PerspectiveCameraType>(null);

  useLayoutEffect(() => {
    const camera = cameraRef.current;
    if (!camera) {
      return;
    }
    camera.lookAt(...walletResponsive.camera.target);
    camera.updateProjectionMatrix();
  }, []);

  return (
    <PerspectiveCamera
      far={50}
      fov={walletResponsive.camera.fov}
      makeDefault
      near={0.1}
      position={[...walletResponsive.camera.position]}
      ref={cameraRef}
    />
  );
}
