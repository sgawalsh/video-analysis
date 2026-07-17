import { useEffect, useState } from 'react';

function VideoTitle({ targetId }) {
  const [title, setTitle] = useState("Loading title...");

  useEffect(() => {
    if (!targetId) return;

    const videoUrl = `https://youtube.com/watch?v=${targetId}`;
    
    // Using noembed's open endpoint to fetch video metadata without an API key
    fetch(`https://noembed.com/embed?dataType=json&url=${videoUrl}`)
      .then((res) => res.json())
      .then((data) => {
        if (data && data.title) {
          setTitle(data.title);
        } else {
          setTitle(`Video ID: ${targetId}`); // Fallback
        }
      })
      .catch(() => {
        setTitle(`Video ID: ${targetId}`); // Fail gracefully
      });
  }, [targetId]);

  return <span>{title}</span>;
}

export default VideoTitle;