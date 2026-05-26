import { useParams } from 'react-router-dom';
import SemanticSearchResult from './semanticSearchResult';

function ResultRouter() {
    const {type, id} = useParams();
    switch (type){
        case 'SEMANTIC_SEARCH':
            return <SemanticSearchResult />
        default:
            return <p>Unknown Result Type</p>
    }
}

export default ResultRouter;