import apiClient from '@/infrastructure/api/http-client';

interface CreateInvitationRequest {
  email: string;
  role: string;
}

interface CreateInvitationResponse {
  id: number;
  token: string;
}

export async function createInvitation(
  email: string,
  role: string,
): Promise<CreateInvitationResponse> {
  return apiClient.post<CreateInvitationResponse>('/invitations', {
    email,
    role,
  } as CreateInvitationRequest);
}
