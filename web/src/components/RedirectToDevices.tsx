import { useParams, Navigate } from 'react-router-dom';

const RedirectToDevices: React.FC = () => {
  const { deviceId } = useParams<{ deviceId: string }>();
  
  if (deviceId) {
    return <Navigate to={`/devices/${deviceId}`} replace />;
  }
  
  return <Navigate to="/devices" replace />;
};

export default RedirectToDevices;