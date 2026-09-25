DELETE FROM company_products
WHERE company_id = '11111111-1111-4111-8111-111111111111'
  AND partner_product_id IN (
      'pizza-margherita',
      'pizza-pepperoni',
      'burger-beef',
      'ramen-chicken',
      'salad-caesar',
      'soup-tom-yum',
      'side-fries',
      'dessert-cheesecake',
      'drink-lemonade',
      'drink-cappuccino'
  );

DELETE FROM users
WHERE id = '22222222-2222-4222-8222-222222222222';

DELETE FROM companies
WHERE id = '11111111-1111-4111-8111-111111111111';
