import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { useMe } from '@/features/auth/hooks/useMe';
import { useUpdateProfile } from '@/features/auth/hooks/useUpdateProfile';
import { useUploadSignature } from '@/features/auth/hooks/useUploadSignature';
import { useSignatureUrl } from '@/features/auth/hooks/useSignatureUrl';

function ProfilePage() {
  const { data: profile, isLoading } = useMe();
  const updateMutation = useUpdateProfile();
  const uploadMutation = useUploadSignature();
  const {
    register,
    handleSubmit,
    reset,
    formState: { isDirty },
  } = useForm();
  const [previewUrl, setPreviewUrl] = useState(null);
  const [selectedFile, setSelectedFile] = useState(null);

  const { data: existingSignatureUrl } = useSignatureUrl(profile?.signature_image_path);

  useEffect(() => {
    if (profile) {
      reset({
        full_name: profile.full_name || '',
        position: profile.position || '',
      });
    }
  }, [profile, reset]);

  const onSubmit = (data) => {
    updateMutation.mutate(
      { full_name: data.full_name, position: data.position },
      {
        onSuccess: () => {
          if (selectedFile) {
            uploadMutation.mutate(selectedFile, {
              onSuccess: () => {
                setSelectedFile(null);
                setPreviewUrl(null);
              },
            });
          } else {
            setSelectedFile(null);
            setPreviewUrl(null);
          }
        },
      },
    );
  };

  const handleFileChange = (e) => {
    const file = e.target.files[0];
    if (file) {
      setSelectedFile(file);
      setPreviewUrl(URL.createObjectURL(file));
    }
  };

  if (isLoading) return <p>Loading...</p>;

  return (
    <div>
      <h1>Profile</h1>
      <form onSubmit={handleSubmit(onSubmit)}>
        <label htmlFor="fullName">Full name</label>
        <input id="fullName" {...register('full_name')} />

        <label htmlFor="position">Position</label>
        <input id="position" {...register('position')} />

        <label htmlFor="signature">Signature</label>
        <input
          id="signature"
          type="file"
          accept="image/png,image/jpeg"
          onChange={handleFileChange}
        />

        {previewUrl && (
          <img data-testid="signature-preview" src={previewUrl} alt="signature preview" />
        )}

        {!previewUrl && existingSignatureUrl && (
          <img data-testid="current-signature" src={existingSignatureUrl} alt="current signature" />
        )}

        {(isDirty || !!selectedFile) && <button type="submit">Save</button>}
      </form>
    </div>
  );
}

export default ProfilePage;
