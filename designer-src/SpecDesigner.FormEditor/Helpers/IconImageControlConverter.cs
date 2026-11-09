using System;
using System.Collections.Generic;
using System.Globalization;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Media.Imaging;
using SpecDesignerCommon.Helpers;

namespace SpecDesigner.FormEditor.Helpers
{
	// Token: 0x02000039 RID: 57
	public class IconImageControlConverter : IValueConverter
	{
		// Token: 0x0600020A RID: 522 RVA: 0x0000AFFC File Offset: 0x000091FC
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			string text = (string)value;
			if (string.IsNullOrEmpty(text))
			{
				return null;
			}
			ComponentType componentType = ComponentType.Item;
			string text2 = "";
			if (!Enum.TryParse<ComponentType>(text, out componentType))
			{
				return null;
			}
			IconImageControlConverter.WidgetImageTable.TryGetValue(componentType, out text2);
			if (this.iconDic.ContainsKey(componentType))
			{
				return this.iconDic[componentType];
			}
			text2 = text2.Replace(".png", "_16.png");
			BitmapImage bitmapImage = IconImageControlConverter.getBitmapImage(text2);
			this.iconDic.Add(componentType, bitmapImage);
			return bitmapImage;
		}

		// Token: 0x0600020B RID: 523 RVA: 0x0000B07E File Offset: 0x0000927E
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}

		// Token: 0x0600020C RID: 524 RVA: 0x0000B088 File Offset: 0x00009288
		private static Image getIcon(string picName)
		{
			return new Image
			{
				Width = 16.0,
				Height = 16.0,
				Source = IconImageControlConverter.getBitmapImage(picName)
			};
		}

		// Token: 0x0600020D RID: 525 RVA: 0x0000B0C6 File Offset: 0x000092C6
		private static BitmapImage getBitmapImage(string picName)
		{
			return new BitmapImage(new Uri("pack://application:,,,/SpecDesigner.FormEditor;component/Images/" + picName));
		}

		// Token: 0x04000124 RID: 292
		private Dictionary<ComponentType, BitmapImage> iconDic = new Dictionary<ComponentType, BitmapImage>();

		// Token: 0x04000125 RID: 293
		private static Dictionary<ComponentType, string> WidgetImageTable = new Dictionary<ComponentType, string>
		{
			{
				ComponentType.Button,
				"widget_button.png"
			},
			{
				ComponentType.ButtonEdit,
				"widget_buttonEdit.png"
			},
			{
				ComponentType.CheckBox,
				"widget_checkbox.png"
			},
			{
				ComponentType.ComboBox,
				"widget_combobox.png"
			},
			{
				ComponentType.DateEdit,
				"widget_dateEdit.png"
			},
			{
				ComponentType.Edit,
				"widget_edit.png"
			},
			{
				ComponentType.FFImage,
				"widget_ffimage.png"
			},
			{
				ComponentType.FFLabel,
				"widget_fflabel.png"
			},
			{
				ComponentType.HLine,
				"widget_hline.png"
			},
			{
				ComponentType.Image,
				"widget_image.png"
			},
			{
				ComponentType.Label,
				"widget_label.png"
			},
			{
				ComponentType.ProgressBar,
				"widget_progressBar.png"
			},
			{
				ComponentType.Phantom,
				"widget_phantom.png"
			},
			{
				ComponentType.Page,
				"widget_page.png"
			},
			{
				ComponentType.RadioGroup,
				"widget_buttonGroup.png"
			},
			{
				ComponentType.Slider,
				"widget_slider.png"
			},
			{
				ComponentType.SpinEdit,
				"widget_spinedit.png"
			},
			{
				ComponentType.TextEdit,
				"widget_textEdit.png"
			},
			{
				ComponentType.TimeEdit,
				"widget_timeedit.png"
			},
			{
				ComponentType.Table,
				"widget_table.png"
			},
			{
				ComponentType.Tree,
				"widget_tree.png"
			},
			{
				ComponentType.WebComponent,
				"widget_webComponent.png"
			},
			{
				ComponentType.Grid,
				"widget_grid.png"
			},
			{
				ComponentType.Group,
				"widget_groupBox.png"
			},
			{
				ComponentType.HBox,
				"edit_hLayout.png"
			},
			{
				ComponentType.VBox,
				"edit_vLayout.png"
			},
			{
				ComponentType.Folder,
				"widget_tab.png"
			},
			{
				ComponentType.ScrollGrid,
				"widget_scrollgrid.png"
			},
			{
				ComponentType.Canvas,
				"widget_canvas.png"
			},
			{
				ComponentType.DateTimeEdit,
				"widget_datetimeEdit.png"
			}
		};
	}
}
