import React from 'react';
import { Composition } from 'remotion';
import { VeloraSplash, type SplashProps } from './VeloraSplash';
import { T_END } from './motion';

const FPS = 60;

export const RemotionRoot: React.FC = () => (
  <Composition
    id="VeloraSplash"
    component={VeloraSplash as React.FC<SplashProps>}
    durationInFrames={Math.ceil(T_END * FPS)}
    fps={FPS}
    width={640}
    height={736}
    defaultProps={{ theme: 'dark' } as SplashProps}
  />
);
