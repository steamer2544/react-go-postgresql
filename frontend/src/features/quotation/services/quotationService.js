import apiClient from '@/services/apiClient';
import { QUOTATIONS } from '@/constants/apiEndpoints';

export async function list(params) {
  return apiClient.get(QUOTATIONS, { params });
}

export async function getById(id) {
  return apiClient.get(`${QUOTATIONS}/${id}`);
}

export async function create(payload) {
  return apiClient.post(QUOTATIONS, payload);
}

export async function update(id, payload) {
  return apiClient.put(`${QUOTATIONS}/${id}`, payload);
}

export async function remove(id) {
  return apiClient.delete(`${QUOTATIONS}/${id}`);
}
