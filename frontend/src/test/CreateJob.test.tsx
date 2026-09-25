import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import App from '../src/App';
import { expect, test, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';

test('user can create a session and see session status', async () => {
  const fetchMock = vi.fn()
    .mockResolvedValueOnce({
      ok: true,
      status: 201,
      json: async () => ({
        public_id: 'test-session-id',
      }),
    })
    .mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({
        counts: {
          total: 1,
          succeeded: 0,
          failed: 0,
          running: 0,
          pending: 1,
        },
        type: 'SEMANTIC_SEARCH',
        errorMessages: [],
        results: [],
        channelSearchStatus: null,
      }),
    });

  const EventSourceMock = vi.fn(function () {
    return {
      addEventListener: vi.fn(),
      close: vi.fn(),
    };
  });

  vi.stubGlobal('EventSource', EventSourceMock);
  vi.stubGlobal('fetch', fetchMock);

  render(
    <MemoryRouter>
        <App />
    </MemoryRouter>
    );

  const user = userEvent.setup();

  await user.type(
    screen.getByPlaceholderText(/Video URL/i),
    'https://www.youtube.com/watch?v=testURL'
  );

  await user.type(
    screen.getByPlaceholderText(/Search term/i),
    'test search'
  );
  
  // Click submit
  const button = screen.getByRole('button', { name: /submit job/i });
  await user.click(button);

  expect(fetchMock).toHaveBeenCalledWith(
    '/api/sessions',
    expect.objectContaining({
      method: 'POST',
    })
  );

  expect(EventSourceMock).toHaveBeenCalledWith('/api/sessions/test-session-id/events');

  expect(await screen.findByText(/Job Status/i)).toBeTruthy();
  expect(await screen.findByRole('heading', { name: /Job Status/i })).toBeTruthy();
});