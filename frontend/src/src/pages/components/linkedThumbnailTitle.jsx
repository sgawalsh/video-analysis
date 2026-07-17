import VideoTitle from "./videoTitle";

function LinkedThumbnailTitle({targetId, startTime=0}){
    return(
        <a
            href={`https://www.youtube.com/watch?v=${targetId}&t=${startTime}`}
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
                <VideoTitle targetId={targetId} />
            </div>
        </a>
    );
}

export default LinkedThumbnailTitle