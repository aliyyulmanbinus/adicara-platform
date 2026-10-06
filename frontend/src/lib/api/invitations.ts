import type { InvitationData } from '@/invitation-templates/types';

interface InvitationResponse {
  id: string;
  event_type: string;
  slug: string;
  title: string;
  template_key: string;
  allow_indexing: boolean;
  design_data?: Record<string, string>;
  media?: Array<{ id: string; kind: string; position: number; url: string }>;
  hosts: Array<{ name: string; role: string }>;
  events: Array<{
    name: string;
    start_at: string;
    end_at?: string;
    timezone: string;
    venue_name: string;
    venue_address: string;
    map_url?: string;
  }>;
}

const defaultSections: InvitationData['sections'] = [
  { id: 'cover', enabled: true, order: 10 },
  { id: 'hosts', enabled: true, order: 20 },
  { id: 'schedule', enabled: true, order: 30 },
  { id: 'closing', enabled: true, order: 40 },
];

function safeExternalURL(value?: string): string | undefined {
  if (!value) return undefined;
  try {
    const url = new URL(value);
    return url.protocol === 'https:' || url.protocol === 'http:'
      ? url.href
      : undefined;
  } catch {
    return undefined;
  }
}

export async function getPublishedInvitation(
  slug: string,
): Promise<InvitationData | null> {
  const apiBaseURL = import.meta.env.API_BASE_URL ?? 'http://localhost:8080';
  const response = await fetch(
    `${apiBaseURL}/api/v1/public/invitations/${encodeURIComponent(slug)}`,
    { headers: { Accept: 'application/json' } },
  );

  if (response.status === 404) return null;
  if (!response.ok)
    throw new Error(`Invitation API returned ${response.status}`);

  const invitation = (await response.json()) as InvitationResponse;
  const media = invitation.media ?? [];
  const mediaURL = (kind: string, position = 0) =>
    media.find((item) => item.kind === kind && item.position === position)?.url;
  return {
    id: invitation.id,
    eventType: invitation.event_type,
    slug: invitation.slug,
    title: invitation.title,
    templateKey: invitation.template_key,
    allowIndexing: invitation.allow_indexing,
    hosts: invitation.hosts.map((host, index) => ({
      ...host,
      photoURL: mediaURL(index === 0 ? 'bride' : 'groom'),
    })),
    galleryImages: media
      .filter((item) => item.kind === 'gallery')
      .sort((a, b) => a.position - b.position)
      .map((item) => item.url),
    designData: invitation.design_data ?? {},
    events: invitation.events.map((event) => ({
      name: event.name,
      startAt: event.start_at,
      endAt: event.end_at,
      timezone: event.timezone,
      venueName: event.venue_name,
      venueAddress: event.venue_address,
      mapURL: safeExternalURL(event.map_url),
    })),
    sections: defaultSections,
  };
}
