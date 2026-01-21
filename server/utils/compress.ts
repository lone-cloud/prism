export function maybeCompress(
  request: Request,
  body: string,
  contentType = 'text/html; charset=utf-8',
): Response {
  const acceptEncoding = request.headers.get('accept-encoding') || '';
  const bodyBytes = new TextEncoder().encode(body);

  if (bodyBytes.length < 1024) {
    return new Response(body, {
      headers: { 'Content-Type': contentType },
    });
  }

  if (acceptEncoding.includes('gzip')) {
    const compressed = Bun.gzipSync(bodyBytes);
    return new Response(compressed, {
      headers: {
        'Content-Type': contentType,
        'Content-Encoding': 'gzip',
      },
    });
  }

  if (acceptEncoding.includes('deflate')) {
    const compressed = Bun.deflateSync(bodyBytes);
    return new Response(compressed, {
      headers: {
        'Content-Type': contentType,
        'Content-Encoding': 'deflate',
      },
    });
  }

  return new Response(body, {
    headers: { 'Content-Type': contentType },
  });
}
