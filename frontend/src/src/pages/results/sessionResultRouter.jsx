import SemanticSearchResult from './semanticSearchResult';
import KeywordSearchResult from './keywordSearchResult';
import TopicDetectionResult from './topicDetectionResult';
import GenericResult from './genericResult';

function SessionResultRouter({ jobType, results }) {
    const renderers = {
        "SEMANTIC_SEARCH": SemanticSearchResult,
        "KEYWORD_SEARCH": KeywordSearchResult,
        "TOPIC_DETECTION_EMBED": TopicDetectionResult,
    };
  const Renderer =
    renderers[jobType] ?? GenericResult;

  return <Renderer results={results} />;
}
export default SessionResultRouter;