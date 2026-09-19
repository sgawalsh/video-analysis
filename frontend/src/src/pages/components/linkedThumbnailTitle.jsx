function LinkedThumbnailTitle({title, targetId, startTime=0}){
    const url = `https://www.youtube.com/watch?v=${targetId}${startTime > 0 ? `&t=\${startTime}` : ''}`;

    return(
        <a
            href={url}
            target="_blank"
            rel="noopener noreferrer"
            >
            <img
                src={`https://img.youtube.com/vi/${targetId}/mqdefault.jpg`}
                alt="thumbnail"
                width={200}
                style={{ borderRadius: 6 }}
            />

            <div style={{ fontSize: 14, fontWeight: "600", marginTop: 8, color: "#333" }}>
                {title}
            </div>
        </a>
    );
}

export default LinkedThumbnailTitle