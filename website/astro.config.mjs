import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import { satteri } from '@astrojs/markdown-satteri';
import { satteriMermaid } from './src/plugins/satteri-mermaid.mjs';

export default defineConfig({
  // TODO: set the production URL once the site is deployed (enables the sitemap).
  site: 'https://cais.example.com',
  markdown: {
    processor: satteri({ mdastPlugins: [satteriMermaid()] }),
  },
  integrations: [
    starlight({
      title: 'cais',
      description: 'A Go TUI for managing Docker Compose stacks',
      defaultLocale: 'en',
      logo: {
        src: './src/assets/cais-icon.svg',
        alt: 'cais',
        replacesTitle: false,
      },
      plugins: [],
      customCss: ['./src/styles/custom.css'],
      sidebar: [
        {
          label: 'User Guide',
          items: [{ autogenerate: { directory: 'users' } }],
        },
        {
          label: 'Contributor Guide',
          items: [{ autogenerate: { directory: 'contributors' } }],
        },
      ],
    }),
  ],
});