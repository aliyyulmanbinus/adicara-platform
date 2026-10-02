import type { InvitationData } from '@/invitation-templates/types';

export const demoInvitation: InvitationData = {
  id: 'preview-adicara',
  eventType: 'wedding',
  slug: 'nara-dan-arka',
  title: 'Pernikahan Nara & Arka',
  templateKey: 'editorial-ivory',
  allowIndexing: false,
  hosts: [
    { name: 'Nara', role: 'Mempelai wanita' },
    { name: 'Arka', role: 'Mempelai pria' },
  ],
  events: [
    {
      name: 'Akad nikah',
      startAt: '2027-12-12T08:00:00+07:00',
      endAt: '2027-12-12T10:00:00+07:00',
      timezone: 'Asia/Jakarta',
      venueName: 'Pendopo Arunika',
      venueAddress: 'Jakarta, Indonesia',
    },
    {
      name: 'Resepsi',
      startAt: '2027-12-12T11:00:00+07:00',
      endAt: '2027-12-12T14:00:00+07:00',
      timezone: 'Asia/Jakarta',
      venueName: 'Pendopo Arunika',
      venueAddress: 'Jakarta, Indonesia',
    },
  ],
  sections: [
    { id: 'cover', enabled: true, order: 10 },
    { id: 'hosts', enabled: true, order: 20 },
    { id: 'schedule', enabled: true, order: 30 },
    { id: 'closing', enabled: true, order: 40 },
  ],
};
