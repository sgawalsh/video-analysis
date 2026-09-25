import { render, screen } from '@testing-library/react';
import { expect, test } from 'vitest';
import { server } from './mocks/server';
import { http, HttpResponse } from 'msw';
import JobsForm from '../src/pages/jobForm';
import { MemoryRouter } from 'react-router-dom';
import { userEvent } from '@testing-library/user-event/dist/cjs/setup/index.js';

server.use(
  http.post('/api/sessions', () =>
    HttpResponse.json(
      { message: 'Internal Server Error' },
      { status: 500 }
    )
  )
);

test('shows error message on API failure', async () => {
  render(
    <MemoryRouter>
      <JobsForm />
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

  await user.click(
    screen.getByRole('button', { name: /submit/i })
  );

  expect(await screen.findByText(/Failed to submit job/i)).toBeTruthy();
});