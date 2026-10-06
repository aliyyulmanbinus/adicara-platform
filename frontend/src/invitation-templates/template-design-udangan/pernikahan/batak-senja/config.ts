import type { InvitationTemplateDefinition } from '../../../types';

export const batakSenjaConfig = {
  key: 'batak-senja',
  name: 'Batak Senja',
  description:
    'Ilustrasi dan ornamen Batak bernuansa senja, disajikan sebagai undangan berhalaman dengan transisi otomatis.',
  category: 'Pernikahan',
  previewImage:
    '/template-design-udangan/pernikahan/batak-senja/landscape.webp',
  supportedSections: ['cover', 'hosts', 'schedule', 'closing'],
  defaultTheme: {
    background: '#8d3e49',
    foreground: '#fff0d5',
    accent: '#d5b47d',
  },
  version: 1,
} satisfies InvitationTemplateDefinition;
