import BotanicalModernTemplate from './botanical-modern/Template.astro';
import { botanicalModernConfig } from './botanical-modern/config';
import EditorialIvoryTemplate from './editorial-ivory/Template.astro';
import { editorialIvoryConfig } from './editorial-ivory/config';
import MonochromeLuxeTemplate from './monochrome-luxe/Template.astro';
import { monochromeLuxeConfig } from './monochrome-luxe/config';

export const invitationTemplates = {
  [editorialIvoryConfig.key]: {
    ...editorialIvoryConfig,
    component: EditorialIvoryTemplate,
  },
  [botanicalModernConfig.key]: {
    ...botanicalModernConfig,
    component: BotanicalModernTemplate,
  },
  [monochromeLuxeConfig.key]: {
    ...monochromeLuxeConfig,
    component: MonochromeLuxeTemplate,
  },
} as const;

export type RegisteredTemplateKey = keyof typeof invitationTemplates;
export type RegisteredTemplate =
  (typeof invitationTemplates)[RegisteredTemplateKey];

export function getInvitationTemplate(key: string): RegisteredTemplate {
  if (key in invitationTemplates) {
    return invitationTemplates[key as RegisteredTemplateKey]!;
  }
  return invitationTemplates[editorialIvoryConfig.key]!;
}
