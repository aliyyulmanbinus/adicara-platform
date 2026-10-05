export type InvitationSectionID = 'cover' | 'hosts' | 'schedule' | 'closing';

export interface InvitationSectionConfig {
  id: InvitationSectionID;
  enabled: boolean;
  order: number;
}

export interface InvitationHost {
  name: string;
  role: string;
  photoURL?: string;
}

export interface InvitationEvent {
  name: string;
  startAt: string;
  endAt?: string;
  timezone: string;
  venueName: string;
  venueAddress: string;
  mapURL?: string;
}

export interface InvitationData {
  id: string;
  eventType: string;
  slug: string;
  title: string;
  templateKey: string;
  allowIndexing: boolean;
  hosts: InvitationHost[];
  events: InvitationEvent[];
  galleryImages?: string[];
  sections: InvitationSectionConfig[];
}

export interface InvitationTemplateDefinition {
  key: string;
  name: string;
  description: string;
  category: string;
  previewImage: string;
  supportedSections: InvitationSectionID[];
  defaultTheme: {
    background: string;
    foreground: string;
    accent: string;
  };
  version: number;
}
