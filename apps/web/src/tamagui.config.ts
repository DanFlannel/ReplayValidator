// Standalone tamagui config. platform-core's apps re-export a shared
// @repo/configs package, but that package lives in a different repo — this
// app has no shared workspace packages yet, so it uses tamagui's own default
// config directly.
import { config as defaultConfig } from '@tamagui/config/v3';
import { createTamagui } from 'tamagui';

// Re-running the default config through createTamagui() gives it the fully
// branded CreateTamaguiConfig type (fontLanguages included). The raw export
// from @tamagui/config/v3 is typed too loosely for TamaguiProvider's prop.
const config = createTamagui(defaultConfig);

type Conf = typeof config;

// Without this, TamaguiProvider's `config` prop falls back to tamagui's
// generic placeholder type, which requires fields (like `fontLanguages`)
// that the concrete config above satisfies at runtime but not structurally.
declare module 'tamagui' {
  interface TamaguiCustomConfig extends Conf {}
}

export default config;
