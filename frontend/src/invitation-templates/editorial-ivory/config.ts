import type { InvitationTemplateDefinition } from '../types';

export const editorialIvoryConfig = {
  key: 'editorial-ivory',
  name: 'Editorial Ivory',
  description:
    'Komposisi editorial yang tenang dengan tipografi klasik dan palet gading hangat.',
  category: 'Editorial',
  previewImage: '/images/editorial-ivory-preview.svg',
  supportedSections: ['cover', 'hosts', 'schedule', 'closing'],
  defaultTheme: {
    background: '#f5f0e7',
    foreground: '#27231f',
    accent: '#8b5f42',
  },
  version: 1,
} satisfies InvitationTemplateDefinition;
