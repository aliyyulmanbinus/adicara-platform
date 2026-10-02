import type { InvitationTemplateDefinition } from '../types';

export const botanicalModernConfig = {
  key: 'botanical-modern',
  name: 'Botanical Modern',
  description:
    'Nuansa hijau alami, bentuk organik, dan susunan modern untuk perayaan yang segar.',
  category: 'Botanical',
  previewImage: '/images/botanical-modern-preview.svg',
  supportedSections: ['cover', 'hosts', 'schedule', 'closing'],
  defaultTheme: {
    background: '#edf1e8',
    foreground: '#183329',
    accent: '#b75f3b',
  },
  version: 1,
} satisfies InvitationTemplateDefinition;
