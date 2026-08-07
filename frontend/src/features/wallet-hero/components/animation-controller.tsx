"use client";

import { useFrame } from "@react-three/fiber";
import { useEffect, useRef, type RefObject } from "react";
import { AnimationMixer, LoopRepeat, type Group } from "three";

import { createWalletAnimationClip } from "../motion/wallet-motion";

export function AnimationController({
  sceneRef,
}: {
  sceneRef: RefObject<Group | null>;
}) {
  const mixerRef = useRef<AnimationMixer | null>(null);

  useFrame((_, delta) => {
    mixerRef.current?.update(delta);
  });

  useEffect(() => {
    const scene = sceneRef.current;
    if (!scene) {
      return;
    }

    const mixer = new AnimationMixer(scene);
    const clip = createWalletAnimationClip();
    const action = mixer.clipAction(clip);
    action.setLoop(LoopRepeat, Infinity);
    action.clampWhenFinished = false;
    action.play();
    mixerRef.current = mixer;

    return () => {
      action.stop();
      mixer.stopAllAction();
      mixer.uncacheClip(clip);
      mixer.uncacheRoot(scene);
      mixerRef.current = null;
    };
  }, [sceneRef]);

  return null;
}
