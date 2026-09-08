import React, { createContext, useContext, useState, useEffect } from 'react';

const LocationContext = createContext(null);

const DEFAULT_COORDS = { lat: 26.1542, lng: 85.8918 }; // Default city coordinates

export const LocationProvider = ({ children }) => {
  const [coords, setCoords] = useState(() => {
    const saved = localStorage.getItem('shopme_customer_coords');
    if (saved) {
      try {
        return JSON.parse(saved);
      } catch (e) {}
    }
    return DEFAULT_COORDS;
  });

  const [locationName, setLocationName] = useState(() => {
    return localStorage.getItem('shopme_customer_loc_name') || 'Local Area';
  });

  const [radiusKm, setRadiusKm] = useState(() => {
    const saved = localStorage.getItem('shopme_customer_radius');
    return saved ? parseFloat(saved) : 5; // Default 5 km radius
  });

  const [isDetecting, setIsDetecting] = useState(false);
  const [gpsError, setGpsError] = useState(null);

  useEffect(() => {
    localStorage.setItem('shopme_customer_coords', JSON.stringify(coords));
  }, [coords]);

  useEffect(() => {
    localStorage.setItem('shopme_customer_loc_name', locationName);
  }, [locationName]);

  useEffect(() => {
    localStorage.setItem('shopme_customer_radius', radiusKm.toString());
  }, [radiusKm]);

  // Detect location via device GPS
  const detectLocation = () => {
    if (!navigator.geolocation) {
      setGpsError('Geolocation is not supported by your browser');
      return;
    }
    setIsDetecting(true);
    setGpsError(null);

    navigator.geolocation.getCurrentPosition(
      (position) => {
        const newCoords = {
          lat: position.coords.latitude,
          lng: position.coords.longitude,
        };
        setCoords(newCoords);
        setLocationName('Current GPS Location');
        setIsDetecting(false);
      },
      (error) => {
        console.warn('GPS location access denied or failed:', error);
        setGpsError('Could not access GPS. Using default area.');
        setIsDetecting(false);
      },
      { timeout: 10000, enableHighAccuracy: true }
    );
  };

  const setManualLocation = (name, lat, lng) => {
    setCoords({ lat, lng });
    setLocationName(name);
  };

  return (
    <LocationContext.Provider
      value={{
        coords,
        locationName,
        radiusKm,
        setRadiusKm,
        detectLocation,
        setManualLocation,
        isDetecting,
        gpsError,
      }}
    >
      {children}
    </LocationContext.Provider>
  );
};

export const useLocation = () => {
  const context = useContext(LocationContext);
  if (!context) {
    throw new Error('useLocation must be used within a LocationProvider');
  }
  return context;
};
