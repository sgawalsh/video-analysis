import SemanticSearchResult from './semanticSearchResult';
import KeywordSearchResult from './keywordSearchResult';
import GenericResult from './genericResult';

function SessionResultRouter({ jobType, results }) {
    const renderers = {
        "SEMANTIC_SEARCH": SemanticSearchResult,
        "KEYWORD_SEARCH": KeywordSearchResult,
    };
  const Renderer =
    renderers[jobType] ?? GenericResult;

  return <Renderer results={results} />;
}
export default SessionResultRouter;