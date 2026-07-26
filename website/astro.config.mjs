// @ts-check
import { defineConfig } from 'astro/config';

import tailwindcss from '@tailwindcss/vite';
import vue from '@astrojs/vue';
import react from '@astrojs/react';

// https://astro.build/config
export default defineConfig({
  site: 'https://1mr0-tech.github.io',
  base: '/logcloak',
  trailingSlash: 'ignore',

  vite: {
    plugins: [tailwindcss()]
  },

  integrations: [vue(), react()]
});