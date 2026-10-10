function LinkedThumbnailTitle({ title, targetId, startTime = 0 }) {
  const url = `https://www.youtube.com/watch?v=${targetId}${
    startTime > 0 ? `&t=${startTime}` : ''
  }`;

  return (
    <a
      href={url}
      target="_blank"
      rel="noopener noreferrer"
      className="linked-thumbnail-title"
    >
      <img
        src={`https://img.youtube.com/vi/${targetId}/mqdefault.jpg`}
        alt="Video thumbnail"
        width={200}
      />
      <div className="linked-thumbnail-text">
        {title}
      </div>
    </a>
  );
}

export default LinkedThumbnailTitle;