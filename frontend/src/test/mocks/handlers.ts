import { http, HttpResponse } from 'msw';

let jobs: any[] = [];

export const handlers = [
  // Create session
  http.post('/api/sessions', async ({ request }) => {
    const body = await request.json() as any;

    if (!body.description) {
      return HttpResponse.json(
        { error: 'Description is required' },
        { status: 400 }
      );
    }

    const job = {
      id: jobs.length + 1,
      description: body.description,
      status: 'PENDING',
    };

    jobs.push(job);

    return HttpResponse.json(job, { status: 201 });
  }),

  // Fetch sessionstatus
  http.get('/api/sessions/:id', ({ params }) => {
    return HttpResponse.json({
      id: Number(params.id),
      description: 'My test job',
      status: 'PENDING',
    });
  }),
];
