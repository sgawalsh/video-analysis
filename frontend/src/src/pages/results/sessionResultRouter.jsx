import SemanticSearchResult from './semanticSearchResult';
import GenericResult from './genericResult';

function SessionResultRouter({ jobType, results }) {
    const renderers = {
        "SEMANTIC_SEARCH": SemanticSearchResult,
    };
  const Renderer =
    renderers[jobType] ?? GenericResult;

  return <Renderer results={results} />;
}
export default SessionResultRouter;