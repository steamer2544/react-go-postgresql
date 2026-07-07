import { useForm } from 'react-hook-form';
import { useNavigate } from 'react-router-dom';
import { useLogin } from '@/features/auth/hooks/useLogin';

function LoginPage() {
  const { register, handleSubmit } = useForm();
  const navigate = useNavigate();
  const loginMutation = useLogin();

  const onSubmit = (data) => {
    loginMutation.mutate(
      { email: data.email, password: data.password },
      {
        onSuccess: () => {
          navigate('/');
        },
      },
    );
  };

  return (
    <div>
      <h1>Log in</h1>
      <form onSubmit={handleSubmit(onSubmit)}>
        <label htmlFor="email">Email</label>
        <input id="email" type="email" {...register('email', { required: true })} />

        <label htmlFor="password">Password</label>
        <input id="password" type="password" {...register('password', { required: true })} />

        {loginMutation.isError && loginMutation.error?.code === 'UNAUTHORIZED' && (
          <div data-testid="login-error">Invalid email or password. Please try again.</div>
        )}

        <button type="submit">Log in</button>
      </form>
    </div>
  );
}

export default LoginPage;
