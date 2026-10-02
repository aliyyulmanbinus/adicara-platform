import type { InvitationTemplateDefinition } from '../types';

export const monochromeLuxeConfig = {
  key: 'monochrome-luxe',
  name: 'Monochrome Luxe',
  description:
    'Kontras hitam-putih, garis presisi, dan tipografi berani untuk suasana kontemporer.',
  category: 'Modern',
  previewImage: '/images/monochrome-luxe-preview.svg',
  supportedSections: ['cover', 'hosts', 'schedule', 'closing'],
  defaultTheme: {
    background: '#f5f5f2',
    foreground: '#111111',
    accent: '#111111',
  },
  version: 1,
} satisfies InvitationTemplateDefinition;
